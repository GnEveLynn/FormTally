package httpapi_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAccessLogsRedactAuthenticationImageAndProfileData(t *testing.T) {
	var logs bytes.Buffer
	router := contractRouter(&logs)
	body := `{"phone":"+8613812345678","code":"839201","heightCm":178.43761,"weightKg":72.59834,"image":"base64-image-secret"}`
	request := httptest.NewRequest(http.MethodPost, "/v1/account-deletions?token=query-token-secret", strings.NewReader(body))
	request.Header.Set("Origin", allowedOrigin)
	request.Header.Set("Authorization", "Bearer authorization-secret")
	request.Header.Set("Cookie", "formtally_session=session-cookie-secret")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	logText := logs.String()
	for _, secret := range []string{"+8613812345678", "839201", "178.43761", "72.59834", "base64-image-secret", "query-token-secret", "authorization-secret", "session-cookie-secret"} {
		if strings.Contains(logText, secret) {
			t.Fatalf("access log leaked %q: %s", secret, logText)
		}
	}
	for _, safeField := range []string{`"method":"POST"`, `"path":"/v1/account-deletions"`, `"status":401`} {
		if !strings.Contains(logText, safeField) {
			t.Fatalf("access log missing %s: %s", safeField, logText)
		}
	}
}
