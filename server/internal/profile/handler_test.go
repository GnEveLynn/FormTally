package profile

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProfileHandlerGetAndPut(t *testing.T) {
	pool := profileTestPool(t)
	if _, err := pool.Exec(context.Background(), `insert into users(id,phone) values('user_http_profile','+8613800000001')`); err != nil {
		t.Fatal(err)
	}
	service := NewService(NewPostgresStore(pool))
	handler := NewHandler(service, func(*http.Request) (string, error) { return "user_http_profile", nil })
	mux := http.NewServeMux()
	handler.Register(mux)

	request := httptest.NewRequest(http.MethodGet, "/v1/profile", nil)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"profile":null`) {
		t.Fatalf("GET status/body = %d %s", response.Code, response.Body.String())
	}

	body := `{"biologicalSex":"male","birthDate":"1995-06-18","heightCm":178,"weightKg":72.5,"activityLevel":"moderate","timezone":"Asia/Shanghai","healthContext":{"pregnant":false,"breastfeeding":false,"clinicalDietRequired":false}}`
	request = httptest.NewRequest(http.MethodPut, "/v1/profile", bytes.NewBufferString(body))
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"revision":1`) || !strings.Contains(response.Body.String(), `"automaticGoalEligible":true`) {
		t.Fatalf("PUT status/body = %d %s", response.Code, response.Body.String())
	}
}
