package httpapi_test

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/GnEveLynn/FormTally/server/internal/account"
	"github.com/GnEveLynn/FormTally/server/internal/analysis"
	"github.com/GnEveLynn/FormTally/server/internal/auth"
	"github.com/GnEveLynn/FormTally/server/internal/days"
	"github.com/GnEveLynn/FormTally/server/internal/goals"
	"github.com/GnEveLynn/FormTally/server/internal/httpapi"
	"github.com/GnEveLynn/FormTally/server/internal/idempotency"
	"github.com/GnEveLynn/FormTally/server/internal/meals"
	"github.com/GnEveLynn/FormTally/server/internal/postgres"
	"github.com/GnEveLynn/FormTally/server/internal/profile"
	"github.com/GnEveLynn/FormTally/server/internal/sms"
	"github.com/GnEveLynn/FormTally/server/internal/storage"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const allowedOrigin = "http://127.0.0.1:5173"

type endpointFixture struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	Body   string `json:"body"`
	Status int    `json:"status"`
}

func TestAllTwentyFourAPIContractsAreRegistered(t *testing.T) {
	data, err := os.ReadFile("testdata/endpoints.json")
	if err != nil {
		t.Fatal(err)
	}
	var endpoints []endpointFixture
	if err := json.Unmarshal(data, &endpoints); err != nil {
		t.Fatal(err)
	}
	if len(endpoints) != 24 {
		t.Fatalf("endpoint fixtures = %d, want 24", len(endpoints))
	}

	router := contractRouter(io.Discard)
	for _, endpoint := range endpoints {
		endpoint := endpoint
		t.Run(endpoint.Method+" "+strings.Split(endpoint.Path, "?")[0], func(t *testing.T) {
			request := httptest.NewRequest(endpoint.Method, endpoint.Path, strings.NewReader(endpoint.Body))
			request.Header.Set("X-Request-ID", "contract-request-123")
			if endpoint.Method != http.MethodGet {
				request.Header.Set("Origin", allowedOrigin)
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			if response.Code != endpoint.Status {
				t.Fatalf("status = %d, want %d; body=%s", response.Code, endpoint.Status, response.Body.String())
			}
			if got := response.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
				t.Fatalf("Content-Type = %q", got)
			}
			if got := response.Header().Get("Cache-Control"); got != "no-store" {
				t.Fatalf("Cache-Control = %q", got)
			}
			var body map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatalf("response is not JSON: %v", err)
			}
			assertCamelCaseKeys(t, body)
			errorBody, ok := body["error"].(map[string]any)
			if !ok || errorBody["requestId"] != "contract-request-123" {
				t.Fatalf("uniform error body = %#v", body)
			}
		})
	}
}

func TestContractEncodingPreservesTimeAndDecimalPrecision(t *testing.T) {
	payload := struct {
		OccurredAt time.Time `json:"occurredAt"`
		WeightKg   float64   `json:"weightKg"`
	}{time.Date(2026, 9, 11, 12, 34, 56, 0, time.FixedZone("CST", 8*60*60)), 72.5}
	response := httptest.NewRecorder()
	httpapi.WriteJSON(response, http.StatusOK, payload)
	if got := strings.TrimSpace(response.Body.String()); got != `{"occurredAt":"2026-09-11T12:34:56+08:00","weightKg":72.5}` {
		t.Fatalf("encoded payload = %s", got)
	}
}

func contractRouter(logOutput io.Writer) http.Handler {
	unauthenticated := func(*http.Request) (string, error) { return "", errors.New("not authenticated") }
	authService := auth.NewService(nil, nil)
	register := []func(*http.ServeMux){
		auth.NewHandler(authService, unauthenticated).Register,
		profile.NewHandler(profile.NewService(nil), unauthenticated).Register,
		goals.NewHandler(goals.NewService(nil), unauthenticated).Register,
		analysis.NewHandler(analysis.NewService(nil, nil, nil, nil), unauthenticated).Register,
		meals.NewHandler(meals.NewService(nil, nil, nil), unauthenticated).Register,
		days.NewHandler(days.NewService(nil, nil), unauthenticated).Register,
		account.NewHandler(account.NewService(nil), unauthenticated).Register,
	}
	return httpapi.NewRouter(slog.New(slog.NewJSONHandler(logOutput, nil)), []string{allowedOrigin}, register...)
}

func assertCamelCaseKeys(t *testing.T, value any) {
	t.Helper()
	switch value := value.(type) {
	case map[string]any:
		for key, child := range value {
			if strings.ContainsRune(key, '_') || (len(key) > 0 && key[0] >= 'A' && key[0] <= 'Z') {
				t.Fatalf("non-camelCase key %q", key)
			}
			assertCamelCaseKeys(t, child)
		}
	case []any:
		for _, child := range value {
			assertCamelCaseKeys(t, child)
		}
	}
}

type fixedAnalyzer struct{}

func (fixedAnalyzer) Analyze(context.Context, []byte, string) (analysis.Result, analysis.Metadata, error) {
	assumption := "按一份熟制鸡肉饭估算"
	return analysis.Result{Items: []analysis.Item{{
		Name: "鸡肉饭", Grams: 350, EnergyKcal: 520, ProteinGrams: 28, CarbGrams: 65, FatGrams: 17,
		Confidence: "medium", Assumption: &assumption,
	}}}, analysis.Metadata{Model: "test-model", PromptVersion: analysis.PromptVersion, ResponseStatus: "completed", Duration: 250 * time.Millisecond}, nil
}

func TestPostgresFullJourneyAcrossModules(t *testing.T) {
	pool := integrationTestPool(t)
	ctx := context.Background()
	now := time.Now().In(time.FixedZone("Asia/Shanghai", 8*60*60)).Truncate(time.Second)
	phone := "+8613812345678"

	sender := sms.NewTestSender(nil)
	authService := auth.NewService(auth.NewPostgresStore(pool), sender)
	verification, err := authService.RequestCode(ctx, auth.RequestCodeInput{Phone: phone, Purpose: auth.PurposeLogin})
	if err != nil {
		t.Fatal(err)
	}
	code, ok := sender.LastCode(phone, string(auth.PurposeLogin))
	if !ok {
		t.Fatal("test SMS sender did not receive a code")
	}
	session, _, err := authService.CreateSession(ctx, auth.CreateSessionInput{
		Phone: phone, Code: code, VerificationRequestID: verification.RequestID,
		TermsVersion: auth.CurrentTermsVersion, PrivacyVersion: auth.CurrentPrivacyVersion,
	})
	if err != nil {
		t.Fatal(err)
	}
	userID := session.User.ID

	goalStore := goals.NewPostgresStore(pool)
	goalService := goals.NewService(goalStore)
	profileService := profile.NewService(profile.NewPostgresStore(pool), goalService)
	if _, err := profileService.Put(ctx, userID, profile.PutInput{
		BiologicalSex: goals.SexMale, BirthDate: "1995-06-18", HeightCm: 178, WeightKg: 72.5,
		ActivityLevel: goals.ActivityModerate, Timezone: "Asia/Shanghai", HealthContext: goals.HealthContext{},
	}); err != nil {
		t.Fatal(err)
	}
	goalResult, err := goalService.Save(ctx, userID, goals.SettingsInput{Mode: goals.ModeAutomatic, Automatic: &goals.AutomaticSettings{Objective: goals.ObjectiveFatLoss, Pace: goals.PaceStandard}})
	if err != nil {
		t.Fatal(err)
	}
	if goalResult.EffectiveTarget.Target.EnergyKcal <= 0 || goalResult.EffectiveTarget.Calculation == nil {
		t.Fatalf("goal result = %+v", goalResult)
	}

	objects := storage.NewFilesystemStore(t.TempDir(), []byte("integration-image-secret"), time.Now)
	idempotencyStore := idempotency.NewPostgresStore(pool)
	analysisService := analysis.NewService(analysis.NewPostgresStore(pool), objects, fixedAnalyzer{}, idempotencyStore)
	occurredAt := now.Add(-time.Minute).Format(time.RFC3339)
	draft, err := analysisService.Create(ctx, userID, "analysis-key", analysis.CreateInput{
		ProcessingMode: "ai", AIConsentVersion: analysis.CurrentAIConsentVersion, OccurredAt: occurredAt,
		MealType: "lunch", Image: []byte("test-image-boundary"), Width: 100, Height: 80,
	})
	if err != nil {
		t.Fatal(err)
	}
	if draft.Status != "review_required" || len(draft.Items) != 1 {
		t.Fatalf("analysis draft = %+v", draft)
	}

	analysisID := draft.ID
	mealService := meals.NewService(meals.NewPostgresStore(pool), idempotencyStore, objects)
	created, err := mealService.Create(ctx, userID, "meal-key", meals.CreateInput{
		AnalysisID: &analysisID, OccurredAt: occurredAt, MealType: "lunch",
		Items: []meals.ItemInput{{Name: "鸡肉饭", Grams: 350, Nutrition: meals.Nutrition{EnergyKcal: 520, ProteinGrams: 28, CarbGrams: 65, FatGrams: 17}, Origin: "ai_modified"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	dayService := days.NewService(days.NewPostgresStore(pool), goalStore)
	day, err := dayService.Get(ctx, userID, created.Meal.LocalDate)
	if err != nil {
		t.Fatal(err)
	}
	if day.Totals.EnergyKcal != 520 || len(day.MealGroups) != 1 {
		t.Fatalf("day after save = %+v", day)
	}

	updatedItems := []meals.ItemInput{{Name: "鸡肉饭", Grams: 360, Nutrition: meals.Nutrition{EnergyKcal: 540, ProteinGrams: 30, CarbGrams: 66, FatGrams: 18}, Origin: "manual"}}
	updated, err := mealService.Update(ctx, userID, created.Meal.ID, meals.UpdateInput{ExpectedRevision: created.Meal.Revision, Items: &updatedItems})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mealService.Update(ctx, userID, created.Meal.ID, meals.UpdateInput{ExpectedRevision: created.Meal.Revision, Items: &updatedItems}); !errors.Is(err, meals.ErrRevisionConflict) {
		t.Fatalf("stale revision error = %v", err)
	}
	history, err := dayService.History(ctx, userID, created.Meal.LocalDate[:7])
	if err != nil || len(history.Days) != 1 || history.Days[0].MealCount != 1 {
		t.Fatalf("history = %+v, err=%v", history, err)
	}
	if _, err := mealService.Get(ctx, "other-user", created.Meal.ID); !errors.Is(err, meals.ErrNotFound) {
		t.Fatalf("cross-user meal error = %v", err)
	}
	if _, err := mealService.Delete(ctx, userID, created.Meal.ID, updated.Meal.Revision); err != nil {
		t.Fatal(err)
	}
	day, err = dayService.Get(ctx, userID, created.Meal.LocalDate)
	if err != nil || day.Totals.EnergyKcal != 0 || len(day.MealGroups) != 0 {
		t.Fatalf("day after delete = %+v, err=%v", day, err)
	}
}

func integrationTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	base := os.Getenv("TEST_DATABASE_URL")
	if base == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	admin, err := postgres.Open(context.Background(), base)
	if err != nil {
		t.Fatal(err)
	}
	database := "test_httpapi_" + strings.ToLower(rand.Text())
	if _, err = admin.Exec(context.Background(), "create database "+pgx.Identifier{database}.Sanitize()); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = admin.Exec(context.Background(), "drop database "+pgx.Identifier{database}.Sanitize()+" with (force)")
		admin.Close()
	})
	parsed, err := url.Parse(base)
	if err != nil {
		t.Fatal(err)
	}
	parsed.Path = "/" + database
	if err := postgres.Migrate(context.Background(), parsed.String(), "../../migrations", "up"); err != nil {
		t.Fatal(err)
	}
	pool, err := postgres.Open(context.Background(), parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}
