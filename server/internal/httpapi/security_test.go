package httpapi_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMutationsRejectMissingAndUntrustedOriginsBeforeHandlers(t *testing.T) {
	router := contractRouter(io.Discard)
	for _, origin := range []string{"", "https://evil.example"} {
		request := httptest.NewRequest(http.MethodPost, "/v1/account-deletions", strings.NewReader(`{"confirmation":"DELETE"}`))
		request.Header.Set("Origin", origin)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusForbidden || !strings.Contains(response.Body.String(), `"code":"ORIGIN_NOT_ALLOWED"`) {
			t.Fatalf("origin %q: status=%d body=%s", origin, response.Code, response.Body.String())
		}
	}
}

func TestProtectedResourcesReturnUniformUnauthenticatedErrors(t *testing.T) {
	router := contractRouter(io.Discard)
	for _, path := range []string{"/v1/profile", "/v1/goals", "/v1/meal-analyses/other", "/v1/meals/other", "/v1/days/2026-09-11", "/v1/history?month=2026-09"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusUnauthorized || !strings.Contains(response.Body.String(), `"code":"UNAUTHENTICATED"`) {
			t.Fatalf("%s: status=%d body=%s", path, response.Code, response.Body.String())
		}
	}
}
