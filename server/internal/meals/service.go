package meals

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/GnEveLynn/FormTally/server/internal/idempotency"
	"github.com/GnEveLynn/FormTally/server/internal/storage"
)

var (
	ErrNotFound         = errors.New("meal not found")
	ErrRevisionConflict = errors.New("meal revision conflict")
	ErrAnalysisExpired  = errors.New("analysis expired")
	ErrAnalysisState    = errors.New("analysis state conflict")
)

type Repository interface {
	Create(context.Context, string, CreateInput, time.Time) (Meal, error)
	Get(context.Context, string, string) (*Meal, error)
	Update(context.Context, string, string, UpdateInput, time.Time) (Meal, []string, error)
	RemoveImage(context.Context, string, string, int, time.Time) (Meal, error)
	Delete(context.Context, string, string, int, time.Time) (DeleteResult, error)
}

type Service struct {
	repo        Repository
	idempotency idempotency.Store
	objects     storage.Store
	now         func() time.Time
}

func NewService(repo Repository, idem idempotency.Store, objects storage.Store) *Service {
	return &Service{repo: repo, idempotency: idem, objects: objects, now: time.Now}
}

func (s *Service) Create(ctx context.Context, userID, key string, input CreateInput) (CreateResult, error) {
	if err := ValidateCreate(input, s.now()); err != nil {
		return CreateResult{}, err
	}
	encoded, _ := json.Marshal(input)
	sum := sha256.Sum256(encoded)
	fingerprint := hex.EncodeToString(sum[:])
	if s.idempotency != nil {
		record, err := s.idempotency.Begin(ctx, userID, "create_meal", key, fingerprint, s.now(), s.now().Add(24*time.Hour))
		if err != nil {
			return CreateResult{}, err
		}
		if record.State == idempotency.Replayed {
			var result CreateResult
			if err = json.Unmarshal(record.Body, &result); err != nil {
				return CreateResult{}, err
			}
			return result, nil
		}
	}
	meal, err := s.repo.Create(ctx, userID, input, s.now())
	if err != nil {
		return CreateResult{}, err
	}
	view, err := s.view(ctx, meal)
	if err != nil {
		return CreateResult{}, err
	}
	result := CreateResult{Meal: view, AffectedLocalDates: []string{meal.LocalDate}}
	if s.idempotency != nil {
		body, _ := json.Marshal(result)
		if err = s.idempotency.Complete(ctx, userID, "create_meal", key, 201, body, s.now()); err != nil {
			return CreateResult{}, err
		}
	}
	return result, nil
}
func (s *Service) Get(ctx context.Context, userID, id string) (MealView, error) {
	meal, err := s.repo.Get(ctx, userID, id)
	if err != nil {
		return MealView{}, err
	}
	if meal == nil {
		return MealView{}, ErrNotFound
	}
	return s.view(ctx, *meal)
}
func (s *Service) Update(ctx context.Context, userID, id string, input UpdateInput) (CreateResult, error) {
	meal, dates, err := s.repo.Update(ctx, userID, id, input, s.now())
	if err != nil {
		return CreateResult{}, err
	}
	view, err := s.view(ctx, meal)
	return CreateResult{Meal: view, AffectedLocalDates: dates}, err
}
func (s *Service) RemoveImage(ctx context.Context, userID, id string, revision int) (MealView, error) {
	meal, err := s.repo.RemoveImage(ctx, userID, id, revision, s.now())
	if err != nil {
		return MealView{}, err
	}
	return s.view(ctx, meal)
}
func (s *Service) Delete(ctx context.Context, userID, id string, revision int) (DeleteResult, error) {
	return s.repo.Delete(ctx, userID, id, revision, s.now())
}
func (s *Service) view(ctx context.Context, meal Meal) (MealView, error) {
	var image *ImageView
	if meal.ImageKey != nil && s.objects != nil {
		url, err := s.objects.PrivateURL(ctx, *meal.ImageKey, 10*time.Minute)
		if err != nil {
			return MealView{}, err
		}
		image = &ImageView{URL: url, ExpiresAt: s.now().Add(10 * time.Minute), MimeType: "image/jpeg"}
	}
	return MealView{ID: meal.ID, OccurredAt: meal.OccurredAt, LocalDate: meal.LocalDate, MealType: meal.MealType, Image: image, Items: meal.Items, Totals: meal.Totals, EstimateNotice: "图片识别和营养数据均为估算，请以实际情况为准。", Revision: meal.Revision, CreatedAt: meal.CreatedAt, UpdatedAt: meal.UpdatedAt}, nil
}
