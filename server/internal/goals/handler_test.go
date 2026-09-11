package goals

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGoalsHandlersPreviewSaveAndGet(t *testing.T) {
	pool := goalsTestPool(t)
	insertGoalTestUser(t, pool, "user_http_goals")
	service := NewService(NewPostgresStore(pool))
	service.now = goalTestNow
	handler := NewHandler(service, func(*http.Request) (string, error) { return "user_http_goals", nil })
	mux := http.NewServeMux()
	handler.Register(mux)
	body := `{"mode":"automatic","automatic":{"objective":"fat_loss","pace":"standard"},"manual":null}`

	request := httptest.NewRequest(http.MethodPost, "/v1/goal-previews", bytes.NewBufferString(body))
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"energyKcal":2220`) {
		t.Fatalf("preview status/body = %d %s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodPut, "/v1/goals", bytes.NewBufferString(body))
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"revision":1`) {
		t.Fatalf("save status/body = %d %s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/v1/goals", nil)
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"activeTarget"`) {
		t.Fatalf("get status/body = %d %s", response.Code, response.Body.String())
	}
}
