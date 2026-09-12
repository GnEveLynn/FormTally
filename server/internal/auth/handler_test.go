package auth

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GnEveLynn/FormTally/server/internal/httpapi"
)

func TestAuthHTTPJourney(t *testing.T) {
	service, sender := testService(t)
	authHandler := NewHandler(service, func(r *http.Request) (string, error) {
		current, err := service.GetSession(r.Context(), SessionToken(r))
		return current.User.ID, err
	})
	router := httpapi.NewRouter(slog.New(slog.NewTextHandler(io.Discard, nil)), []string{"http://127.0.0.1:5173"}, authHandler.Register)
	phone := "+8613812345678"

	codeResponse := performJSON(router, http.MethodPost, "/v1/auth/codes", `{"phone":"+8613812345678","purpose":"login"}`, "", true)
	if codeResponse.Code != http.StatusAccepted || strings.Contains(codeResponse.Body.String(), "123456") {
		t.Fatalf("code response = %d %s", codeResponse.Code, codeResponse.Body.String())
	}
	var codeBody struct {
		Verification Verification `json:"verification"`
	}
	if err := json.Unmarshal(codeResponse.Body.Bytes(), &codeBody); err != nil {
		t.Fatal(err)
	}
	code, _ := sender.LastCode(phone, string(PurposeLogin))
	loginBody := `{"phone":"` + phone + `","code":"` + code + `","verificationRequestId":"` + codeBody.Verification.RequestID + `","agreements":{"termsVersion":"2026-09-10","privacyVersion":"2026-09-10"}}`
	loginResponse := performJSON(router, http.MethodPost, "/v1/auth/sessions", loginBody, "", true)
	if loginResponse.Code != http.StatusCreated {
		t.Fatalf("login response = %d %s", loginResponse.Code, loginResponse.Body.String())
	}
	cookie := loginResponse.Result().Cookies()[0]
	if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("cookie = %+v", cookie)
	}
	current := performJSON(router, http.MethodGet, "/v1/auth/session", "", cookie.Value, false)
	if current.Code != http.StatusOK {
		t.Fatalf("current session = %d %s", current.Code, current.Body.String())
	}
	deleteCodeWithoutSession := performJSON(router, http.MethodPost, "/v1/auth/codes", `{"phone":"+8613812345678","purpose":"delete_account"}`, "", true)
	if deleteCodeWithoutSession.Code != http.StatusUnauthorized {
		t.Fatalf("delete code without session = %d %s", deleteCodeWithoutSession.Code, deleteCodeWithoutSession.Body.String())
	}
	deleteCodeWrongPhone := performJSON(router, http.MethodPost, "/v1/auth/codes", `{"phone":"+8613912345678","purpose":"delete_account"}`, cookie.Value, true)
	if deleteCodeWrongPhone.Code != http.StatusUnprocessableEntity {
		t.Fatalf("delete code wrong phone = %d %s", deleteCodeWrongPhone.Code, deleteCodeWrongPhone.Body.String())
	}
	deleteCode := performJSON(router, http.MethodPost, "/v1/auth/codes", `{"phone":"+8613812345678","purpose":"delete_account"}`, cookie.Value, true)
	if deleteCode.Code != http.StatusAccepted {
		t.Fatalf("delete code current phone = %d %s", deleteCode.Code, deleteCode.Body.String())
	}
	logout := performJSON(router, http.MethodDelete, "/v1/auth/session", "", cookie.Value, true)
	if logout.Code != http.StatusNoContent || logout.Result().Cookies()[0].MaxAge >= 0 {
		t.Fatalf("logout = %d, cookies = %+v", logout.Code, logout.Result().Cookies())
	}
	after := performJSON(router, http.MethodGet, "/v1/auth/session", "", cookie.Value, false)
	if after.Code != http.StatusUnauthorized {
		t.Fatalf("after logout = %d %s", after.Code, after.Body.String())
	}
}

func performJSON(handler http.Handler, method, path, body, token string, withOrigin bool) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json; charset=utf-8")
	if withOrigin {
		request.Header.Set("Origin", "http://127.0.0.1:5173")
	}
	if token != "" {
		request.AddCookie(&http.Cookie{Name: SessionCookieName, Value: token})
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
