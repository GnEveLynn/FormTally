package analysis

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/GnEveLynn/FormTally/server/internal/idempotency"
)

type fakeAnalyzer struct {
	calls       int
	description string
	result      Result
	err         error
}

func (f *fakeAnalyzer) Analyze(_ context.Context, _ []byte, description string) (Result, Metadata, error) {
	f.calls++
	f.description = description
	return f.result, Metadata{Model: "fake", PromptVersion: PromptVersion}, f.err
}

func TestCreateAnalysisPassesOneFreeformDescriptionToAI(t *testing.T) {
	analyzer := &fakeAnalyzer{result: Result{Items: []Item{{Name: "米饭", Grams: 100, EnergyKcal: 116, Confidence: "high"}}}}
	service := NewService(&memoryDrafts{}, &memoryImages{}, analyzer, nil)
	service.now = func() time.Time { return time.Date(2026, 9, 15, 8, 0, 0, 0, time.UTC) }
	_, err := service.Create(context.Background(), "user_1", "key", CreateInput{ProcessingMode: "ai", AIConsentVersion: CurrentAIConsentVersion, OccurredAt: "2026-09-15T12:00:00+08:00", MealType: "lunch", Description: "鸡胸肉和米饭，少油", Image: []byte{1}, Width: 1, Height: 1})
	if err != nil {
		t.Fatal(err)
	}
	if analyzer.description != "鸡胸肉和米饭，少油" {
		t.Fatalf("description = %q", analyzer.description)
	}
}

type memoryDrafts struct {
	drafts  map[string]Draft
	consent string
}

func (m *memoryDrafts) Insert(_ context.Context, draft Draft) error {
	if m.drafts == nil {
		m.drafts = map[string]Draft{}
	}
	m.drafts[draft.ID] = draft
	return nil
}
func (m *memoryDrafts) Update(_ context.Context, draft Draft) error {
	m.drafts[draft.ID] = draft
	return nil
}
func (m *memoryDrafts) Get(_ context.Context, userID, id string) (*Draft, error) {
	d, ok := m.drafts[id]
	if !ok || d.UserID != userID {
		return nil, nil
	}
	return &d, nil
}
func (m *memoryDrafts) Discard(_ context.Context, userID, id string, revision int) (string, error) {
	d, ok := m.drafts[id]
	if !ok || d.UserID != userID {
		return "", nil
	}
	if d.Revision != revision {
		return "", ErrRevisionConflict
	}
	delete(m.drafts, id)
	return d.ImageKey, nil
}
func (m *memoryDrafts) SaveConsent(_ context.Context, _ string, version string, _ time.Time) error {
	m.consent = version
	return nil
}

type memoryImages struct {
	puts, deletes int
	data          []byte
}

func (m *memoryImages) Put(context.Context, string, io.Reader, string) (string, error) {
	m.puts++
	return "0123456789abcdef0123456789abcdef0123456789abcdef", nil
}
func (m *memoryImages) Open(context.Context, string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(m.data)), nil
}
func (m *memoryImages) PrivateURL(context.Context, string, time.Duration) (string, error) {
	return "/private/image", nil
}
func (m *memoryImages) Delete(context.Context, string) error { m.deletes++; return nil }

type memoryIdempotency struct{ records map[string]idempotency.Result }

func (m *memoryIdempotency) Begin(_ context.Context, _, operation, key, _ string, _, _ time.Time) (idempotency.Result, error) {
	if result, ok := m.records[operation+key]; ok {
		return result, nil
	}
	if m.records == nil {
		m.records = map[string]idempotency.Result{}
	}
	return idempotency.Result{State: idempotency.Started}, nil
}
func (m *memoryIdempotency) Complete(_ context.Context, _, operation, key string, status int, body []byte, _ time.Time) error {
	m.records[operation+key] = idempotency.Result{State: idempotency.Replayed, Status: status, Body: body}
	return nil
}

func TestCreateAnalysisRequiresCurrentConsentBeforeCallingAI(t *testing.T) {
	repo, images, analyzer := &memoryDrafts{}, &memoryImages{}, &fakeAnalyzer{}
	service := NewService(repo, images, analyzer, nil)
	service.now = func() time.Time { return time.Date(2026, 9, 11, 8, 0, 0, 0, time.UTC) }
	_, err := service.Create(context.Background(), "user_1", "key", CreateInput{ProcessingMode: "ai", AIConsentVersion: "old", OccurredAt: "2026-09-11T12:00:00+08:00", MealType: "lunch", Image: []byte{1}, Width: 1, Height: 1})
	if !errors.Is(err, ErrAIConsentRequired) || analyzer.calls != 0 || images.puts != 0 {
		t.Fatalf("err=%v analyzer calls=%d image puts=%d", err, analyzer.calls, images.puts)
	}
}

func TestCreateAnalysisPersistsRecoverableFailedResource(t *testing.T) {
	repo, images := &memoryDrafts{}, &memoryImages{}
	analyzer := &fakeAnalyzer{err: &Failure{Code: "AI_TIMEOUT", Retryable: true, Err: context.DeadlineExceeded}}
	service := NewService(repo, images, analyzer, nil)
	service.now = func() time.Time { return time.Date(2026, 9, 11, 8, 0, 0, 0, time.UTC) }
	view, err := service.Create(context.Background(), "user_1", "key", CreateInput{ProcessingMode: "ai", AIConsentVersion: CurrentAIConsentVersion, OccurredAt: "2026-09-11T12:00:00+08:00", MealType: "lunch", Image: []byte{1}, Width: 1, Height: 1})
	if err != nil {
		t.Fatal(err)
	}
	if view.Status != "failed" || view.Failure == nil || view.Failure.Code != "AI_TIMEOUT" || view.ExpiresAt.Sub(view.CreatedAt) != 24*time.Hour {
		t.Fatalf("view = %+v", view)
	}
}

func TestCreateAnalysisReturnsMealItemsInPublicAPIShape(t *testing.T) {
	analyzer := &fakeAnalyzer{result: Result{Items: []Item{{
		Name: "米饭", Grams: 100, EnergyKcal: 116, ProteinGrams: 2.6, CarbGrams: 25.9, FatGrams: .3, Confidence: "high",
	}}}}
	service := NewService(&memoryDrafts{}, &memoryImages{}, analyzer, nil)
	service.now = func() time.Time { return time.Date(2026, 9, 11, 8, 0, 0, 0, time.UTC) }
	view, err := service.Create(context.Background(), "user_1", "key", CreateInput{ProcessingMode: "ai", AIConsentVersion: CurrentAIConsentVersion, OccurredAt: "2026-09-11T12:00:00+08:00", MealType: "lunch", Image: []byte{1}, Width: 1, Height: 1})
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(view.Items[0])
	if err != nil {
		t.Fatal(err)
	}
	var item map[string]any
	if err := json.Unmarshal(body, &item); err != nil {
		t.Fatal(err)
	}
	nutrition, ok := item["nutrition"].(map[string]any)
	_, hasDraftItemID := item["draftItemId"]
	if !ok || nutrition["energyKcal"] != float64(116) || item["origin"] != "ai" || !hasDraftItemID {
		t.Fatalf("public item = %s", body)
	}
	if _, leaked := item["energyKcal"]; leaked {
		t.Fatalf("internal nutrition field leaked into public item: %s", body)
	}
}

func TestManualAnalysisSkipsAIAndCreatesEmptyReviewDraft(t *testing.T) {
	repo, images, analyzer := &memoryDrafts{}, &memoryImages{}, &fakeAnalyzer{}
	service := NewService(repo, images, analyzer, nil)
	service.now = func() time.Time { return time.Date(2026, 9, 11, 8, 0, 0, 0, time.UTC) }
	view, err := service.Create(context.Background(), "user_1", "key", CreateInput{ProcessingMode: "manual", OccurredAt: "2026-09-11T12:00:00+08:00", MealType: "lunch", Image: []byte{1}, Width: 1, Height: 1})
	if err != nil {
		t.Fatal(err)
	}
	if view.Status != "review_required" || len(view.Items) != 0 || analyzer.calls != 0 {
		t.Fatalf("view=%+v calls=%d", view, analyzer.calls)
	}
}

func TestRetryAnalysisReplaysWithoutCallingAnalyzerAgain(t *testing.T) {
	now := time.Date(2026, 9, 11, 8, 0, 0, 0, time.UTC)
	repo := &memoryDrafts{drafts: map[string]Draft{"analysis_1": {ID: "analysis_1", UserID: "user_1", ImageKey: "image_1", Status: "failed", ProcessingMode: "ai", OccurredAt: now, LocalDate: "2026-09-11", MealType: "lunch", ExpiresAt: now.Add(time.Hour), Revision: 1, CreatedAt: now}}}
	analyzer := &fakeAnalyzer{result: Result{Items: []Item{{Name: "米饭", Grams: 100, EnergyKcal: 116, Confidence: "high"}}}}
	service := NewService(repo, &memoryImages{data: []byte{1}}, analyzer, &memoryIdempotency{})
	service.now = func() time.Time { return now }
	first, err := service.Retry(context.Background(), "user_1", "analysis_1", "retry-key", CurrentAIConsentVersion, 1)
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Retry(context.Background(), "user_1", "analysis_1", "retry-key", CurrentAIConsentVersion, 1)
	if err != nil {
		t.Fatal(err)
	}
	if analyzer.calls != 1 || first.Revision != second.Revision || first.Status != "review_required" {
		t.Fatalf("calls=%d first=%+v second=%+v", analyzer.calls, first, second)
	}
}
