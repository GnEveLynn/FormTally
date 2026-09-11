package meals

import (
	"bytes"
	"context"
	"github.com/GnEveLynn/FormTally/server/internal/httpapi"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type handlerRepo struct{}

func (handlerRepo) Create(_ context.Context, user string, input CreateInput, now time.Time) (Meal, error) {
	occurred, _ := time.Parse(time.RFC3339, input.OccurredAt)
	items := []Item{{ID: "item_1", ItemInput: input.Items[0]}}
	return Meal{ID: "meal_1", UserID: user, OccurredAt: occurred, LocalDate: "2026-09-11", MealType: input.MealType, Items: items, Totals: Sum(items), Revision: 1, CreatedAt: now, UpdatedAt: now}, nil
}
func (handlerRepo) Get(context.Context, string, string) (*Meal, error) { return nil, nil }
func (handlerRepo) Update(context.Context, string, string, UpdateInput, time.Time) (Meal, []string, error) {
	return Meal{}, nil, nil
}
func (handlerRepo) RemoveImage(context.Context, string, string, int, time.Time) (Meal, error) {
	return Meal{}, nil
}
func (handlerRepo) Delete(context.Context, string, string, int, time.Time) (DeleteResult, error) {
	return DeleteResult{}, nil
}
func TestMealHandlerCreatesFromFinalEditedValues(t *testing.T) {
	service := NewService(handlerRepo{}, nil, nil)
	service.now = func() time.Time { return time.Date(2026, 9, 11, 8, 0, 0, 0, time.UTC) }
	handler := NewHandler(service, func(*http.Request) (string, error) { return "user_1", nil })
	router := httpapi.NewRouter(slog.New(slog.NewTextHandler(io.Discard, nil)), []string{"http://127.0.0.1:5173"}, handler.Register)
	body := `{"analysisId":null,"occurredAt":"2026-09-11T12:00:00+08:00","mealType":"lunch","items":[{"draftItemId":null,"name":"米饭","grams":100,"nutrition":{"energyKcal":200,"proteinGrams":3,"carbGrams":40,"fatGrams":1},"basisPer100Grams":null,"origin":"manual","confidence":null,"assumption":null}]}`
	request := httptest.NewRequest(http.MethodPost, "/v1/meals", bytes.NewBufferString(body))
	request.Header.Set("Origin", "http://127.0.0.1:5173")
	request.Header.Set("Idempotency-Key", "save_1")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != 201 || !strings.Contains(response.Body.String(), `"energyKcal":200`) {
		t.Fatalf("response=%d %s", response.Code, response.Body.String())
	}
}
