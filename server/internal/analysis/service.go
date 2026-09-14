package analysis

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/GnEveLynn/FormTally/server/internal/auth"
	"github.com/GnEveLynn/FormTally/server/internal/idempotency"
	"github.com/GnEveLynn/FormTally/server/internal/storage"
)

const CurrentAIConsentVersion = auth.CurrentAIImageProcessingVersion

var (
	ErrAIConsentRequired = errors.New("current AI consent required")
	ErrInvalid           = errors.New("invalid analysis request")
	ErrExpired           = errors.New("analysis expired")
	ErrStateConflict     = errors.New("analysis state conflict")
	ErrRevisionConflict  = errors.New("analysis revision conflict")
)

type FailureView struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}
type ImageView struct {
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expiresAt"`
	Width     int       `json:"width"`
	Height    int       `json:"height"`
	MimeType  string    `json:"mimeType"`
}
type ItemView struct {
	DraftItemID      *string    `json:"draftItemId"`
	Name             string     `json:"name"`
	Grams            float64    `json:"grams"`
	Nutrition        Nutrition  `json:"nutrition"`
	BasisPer100Grams *Nutrition `json:"basisPer100Grams"`
	Origin           string     `json:"origin"`
	Confidence       *string    `json:"confidence"`
	Assumption       *string    `json:"assumption"`
}
type View struct {
	ID             string       `json:"id"`
	Status         string       `json:"status"`
	ProcessingMode string       `json:"processingMode"`
	OccurredAt     time.Time    `json:"occurredAt"`
	LocalDate      string       `json:"localDate"`
	MealType       string       `json:"mealType"`
	Image          ImageView    `json:"image"`
	Items          []ItemView   `json:"items"`
	Warnings       []string     `json:"warnings"`
	Failure        *FailureView `json:"failure"`
	MealID         *string      `json:"mealId"`
	ExpiresAt      time.Time    `json:"expiresAt"`
	Revision       int          `json:"revision"`
	CreatedAt      time.Time    `json:"createdAt"`
	Metadata       Metadata     `json:"-"`
}
type Draft struct {
	ID, UserID, ImageKey, ProcessingMode, LocalDate, MealType, Status string
	ImageWidth, ImageHeight                                           int
	OccurredAt                                                        time.Time
	Result                                                            Result
	Failure                                                           *FailureView
	Metadata                                                          Metadata
	MealID                                                            *string
	ExpiresAt                                                         time.Time
	Revision                                                          int
	CreatedAt, UpdatedAt                                              time.Time
}
type CreateInput struct {
	ProcessingMode, AIConsentVersion, OccurredAt, MealType string
	Image                                                  []byte
	Width, Height                                          int
}

type Repository interface {
	Insert(context.Context, Draft) error
	Update(context.Context, Draft) error
	Get(context.Context, string, string) (*Draft, error)
	Discard(context.Context, string, string, int) (string, error)
	SaveConsent(context.Context, string, string, time.Time) error
}

type Service struct {
	repo        Repository
	objects     storage.Store
	analyzer    Analyzer
	idempotency idempotency.Store
	now         func() time.Time
}

func NewService(repo Repository, objects storage.Store, analyzer Analyzer, idem idempotency.Store) *Service {
	return &Service{repo: repo, objects: objects, analyzer: analyzer, idempotency: idem, now: time.Now}
}

func (s *Service) Create(ctx context.Context, userID, key string, input CreateInput) (View, error) {
	if s.idempotency != nil && key == "" {
		return View{}, ErrInvalid
	}
	if input.ProcessingMode == "ai" && input.AIConsentVersion != CurrentAIConsentVersion {
		return View{}, ErrAIConsentRequired
	}
	occurredAt, err := s.validate(input)
	if err != nil {
		return View{}, err
	}
	fingerprint := createFingerprint(input)
	if s.idempotency != nil {
		result, err := s.idempotency.Begin(ctx, userID, "create_analysis", key, fingerprint, s.now(), s.now().Add(24*time.Hour))
		if err != nil {
			return View{}, err
		}
		if result.State == idempotency.Replayed {
			var view View
			if err := json.Unmarshal(result.Body, &view); err != nil {
				return View{}, err
			}
			return view, nil
		}
	}
	imageKey, err := s.objects.Put(ctx, userID, bytes.NewReader(input.Image), "image/jpeg")
	if err != nil {
		return View{}, err
	}
	now := s.now()
	draft := Draft{ID: "analysis_" + rand.Text(), UserID: userID, ImageKey: imageKey, ImageWidth: input.Width, ImageHeight: input.Height, ProcessingMode: input.ProcessingMode, OccurredAt: occurredAt, LocalDate: occurredAt.Format("2006-01-02"), MealType: input.MealType, Status: "processing", Result: Result{Items: []Item{}}, ExpiresAt: now.Add(24 * time.Hour), Revision: 1, CreatedAt: now, UpdatedAt: now}
	if err := s.repo.Insert(ctx, draft); err != nil {
		_ = s.objects.Delete(ctx, imageKey)
		return View{}, err
	}
	if input.ProcessingMode == "manual" {
		draft.Status = "review_required"
	} else {
		if err := s.repo.SaveConsent(ctx, userID, input.AIConsentVersion, now); err != nil {
			return View{}, err
		}
		result, meta, analyzeErr := s.analyzer.Analyze(ctx, input.Image)
		draft.Metadata = meta
		if analyzeErr != nil {
			draft.Status = "failed"
			draft.Failure = analysisFailure(analyzeErr)
		} else {
			draft.Status = "review_required"
			draft.Result = result
		}
	}
	draft.UpdatedAt = s.now()
	if err := s.repo.Update(ctx, draft); err != nil {
		return View{}, err
	}
	view, err := s.view(ctx, draft)
	if err != nil {
		return View{}, err
	}
	if s.idempotency != nil {
		body, _ := json.Marshal(view)
		if err := s.idempotency.Complete(ctx, userID, "create_analysis", key, http.StatusCreated, body, s.now()); err != nil {
			return View{}, err
		}
	}
	return view, nil
}

func (s *Service) Get(ctx context.Context, userID, id string) (View, error) {
	draft, err := s.repo.Get(ctx, userID, id)
	if err != nil || draft == nil {
		return View{}, err
	}
	if draft.Status != "saved" && !draft.ExpiresAt.After(s.now()) {
		return View{}, ErrExpired
	}
	return s.view(ctx, *draft)
}

func (s *Service) Retry(ctx context.Context, userID, id, key, consent string, expectedRevision int) (View, error) {
	if consent != CurrentAIConsentVersion {
		return View{}, ErrAIConsentRequired
	}
	if s.idempotency != nil && key == "" {
		return View{}, ErrInvalid
	}
	draft, err := s.repo.Get(ctx, userID, id)
	if err != nil || draft == nil {
		return View{}, err
	}
	if !draft.ExpiresAt.After(s.now()) {
		return View{}, ErrExpired
	}
	if s.idempotency != nil {
		fingerprint := retryFingerprint(id, consent, expectedRevision)
		result, beginErr := s.idempotency.Begin(ctx, userID, "retry_analysis", key, fingerprint, s.now(), s.now().Add(24*time.Hour))
		if beginErr != nil {
			return View{}, beginErr
		}
		if result.State == idempotency.Replayed {
			var view View
			if err := json.Unmarshal(result.Body, &view); err != nil {
				return View{}, err
			}
			return view, nil
		}
	}
	if draft.Revision != expectedRevision {
		return View{}, ErrRevisionConflict
	}
	if draft.Status != "failed" {
		return View{}, ErrStateConflict
	}
	reader, err := s.objects.Open(ctx, draft.ImageKey)
	if err != nil {
		return View{}, err
	}
	defer reader.Close()
	image, err := io.ReadAll(reader)
	if err != nil {
		return View{}, err
	}
	draft.Status, draft.Failure, draft.Revision = "processing", nil, draft.Revision+1
	if err := s.repo.Update(ctx, *draft); err != nil {
		return View{}, err
	}
	result, meta, analyzeErr := s.analyzer.Analyze(ctx, image)
	draft.Metadata = meta
	if analyzeErr != nil {
		draft.Status = "failed"
		draft.Failure = analysisFailure(analyzeErr)
	} else {
		draft.Status = "review_required"
		draft.Result = result
	}
	draft.UpdatedAt = s.now()
	if err := s.repo.Update(ctx, *draft); err != nil {
		return View{}, err
	}
	view, err := s.view(ctx, *draft)
	if err != nil {
		return View{}, err
	}
	if s.idempotency != nil {
		body, _ := json.Marshal(view)
		if err := s.idempotency.Complete(ctx, userID, "retry_analysis", key, http.StatusOK, body, s.now()); err != nil {
			return View{}, err
		}
	}
	return view, nil
}

func (s *Service) Discard(ctx context.Context, userID, id string, revision int) error {
	key, err := s.repo.Discard(ctx, userID, id, revision)
	if err != nil || key == "" {
		return err
	}
	return s.objects.Delete(ctx, key)
}

func (s *Service) validate(input CreateInput) (time.Time, error) {
	if (input.ProcessingMode != "ai" && input.ProcessingMode != "manual") || (input.MealType != "breakfast" && input.MealType != "lunch" && input.MealType != "dinner" && input.MealType != "snack") || len(input.Image) == 0 || input.Width < 1 || input.Height < 1 {
		return time.Time{}, ErrInvalid
	}
	occurred, err := time.Parse(time.RFC3339, input.OccurredAt)
	if err != nil || occurred.After(s.now().Add(time.Minute)) {
		return time.Time{}, ErrInvalid
	}
	return occurred, nil
}

func (s *Service) view(ctx context.Context, draft Draft) (View, error) {
	url, err := s.objects.PrivateURL(ctx, draft.ImageKey, 10*time.Minute)
	if err != nil {
		return View{}, err
	}
	warnings := []string{}
	if draft.Result.Warning != nil {
		warnings = append(warnings, *draft.Result.Warning)
	}
	items := make([]ItemView, len(draft.Result.Items))
	for index, item := range draft.Result.Items {
		confidence, basis := item.Confidence, item.BasisPer100Grams
		items[index] = ItemView{DraftItemID: nil, Name: item.Name, Grams: item.Grams, Nutrition: Nutrition{EnergyKcal: item.EnergyKcal, ProteinGrams: item.ProteinGrams, CarbGrams: item.CarbGrams, FatGrams: item.FatGrams}, BasisPer100Grams: &basis, Origin: "ai", Confidence: &confidence, Assumption: item.Assumption}
	}
	return View{ID: draft.ID, Status: draft.Status, ProcessingMode: draft.ProcessingMode, OccurredAt: draft.OccurredAt, LocalDate: draft.LocalDate, MealType: draft.MealType, Image: ImageView{URL: url, ExpiresAt: s.now().Add(10 * time.Minute), Width: draft.ImageWidth, Height: draft.ImageHeight, MimeType: "image/jpeg"}, Items: items, Warnings: warnings, Failure: draft.Failure, MealID: draft.MealID, ExpiresAt: draft.ExpiresAt, Revision: draft.Revision, CreatedAt: draft.CreatedAt, Metadata: draft.Metadata}, nil
}

func analysisFailure(err error) *FailureView {
	var failure *Failure
	if errors.As(err, &failure) {
		messages := map[string]string{"AI_TIMEOUT": "分析超时，可以使用当前图片重试或手动录入", "FOOD_NOT_RECOGNIZED": "没有可靠识别出食物，可以重拍或手动录入", "AI_OUTPUT_INVALID": "分析结果无效，可以重试或手动录入", "AI_UNAVAILABLE": "分析服务暂时不可用，可以稍后重试"}
		return &FailureView{Code: failure.Code, Message: messages[failure.Code], Retryable: failure.Retryable}
	}
	return &FailureView{Code: "AI_UNAVAILABLE", Message: "分析服务暂时不可用，可以稍后重试", Retryable: true}
}
func createFingerprint(input CreateInput) string {
	sum := sha256.Sum256(append([]byte(fmt.Sprintf("%s|%s|%s|%s|", input.ProcessingMode, input.AIConsentVersion, input.OccurredAt, input.MealType)), input.Image...))
	return hex.EncodeToString(sum[:])
}

func retryFingerprint(id, consent string, revision int) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%d", id, consent, revision)))
	return hex.EncodeToString(sum[:])
}
