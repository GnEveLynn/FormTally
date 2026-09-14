package auth

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GnEveLynn/FormTally/server/internal/httpapi"
)

func TestSessionCredential(t *testing.T) {
	tests := []struct {
		name      string
		configure func(*http.Request)
		want      Credential
		wantError bool
	}{
		{
			name: "cookie",
			configure: func(request *http.Request) {
				request.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "cookie-token"})
			},
			want: Credential{Token: "cookie-token", Kind: CredentialCookie},
		},
		{
			name: "bearer",
			configure: func(request *http.Request) {
				request.Header.Set("Authorization", "Bearer bearer-token")
			},
			want: Credential{Token: "bearer-token", Kind: CredentialBearer},
		},
		{
			name: "wrong scheme",
			configure: func(request *http.Request) {
				request.Header.Set("Authorization", "Basic abc")
			},
			wantError: true,
		},
		{
			name: "empty bearer",
			configure: func(request *http.Request) {
				request.Header.Set("Authorization", "Bearer")
			},
			wantError: true,
		},
		{
			name: "duplicate authorization",
			configure: func(request *http.Request) {
				request.Header.Add("Authorization", "Bearer first")
				request.Header.Add("Authorization", "Bearer second")
			},
			wantError: true,
		},
		{
			name: "cookie and bearer",
			configure: func(request *http.Request) {
				request.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "cookie-token"})
				request.Header.Set("Authorization", "Bearer bearer-token")
			},
			wantError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/v1/auth/session", nil)
			test.configure(request)
			credential, err := SessionCredential(request)
			if test.wantError {
				if err == nil {
					t.Fatalf("SessionCredential() = %+v, nil", credential)
				}
				return
			}
			if err != nil || credential != test.want {
				t.Fatalf("SessionCredential() = %+v, %v", credential, err)
			}
		})
	}
}

func TestBearerCannotBypassOriginOnBrowserAuthEndpoints(t *testing.T) {
	register := func(mux *http.ServeMux) {
		for _, path := range []string{"/v1/auth/codes", "/v1/auth/sessions", "/v1/auth/wechat/sessions", "/v1/auth/wechat/phone-bindings", "/v1/resource"} {
			mux.HandleFunc("POST "+path, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
		}
	}
	handler := WithSessionCredential(httpapi.NewRouter(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		[]string{"http://127.0.0.1:5173"},
		register,
	))

	tests := []struct {
		name   string
		path   string
		origin string
		want   int
	}{
		{name: "codes require origin despite bearer", path: "/v1/auth/codes", want: http.StatusForbidden},
		{name: "browser sessions require origin despite bearer", path: "/v1/auth/sessions", want: http.StatusForbidden},
		{name: "browser auth accepts allowed origin", path: "/v1/auth/codes", origin: "http://127.0.0.1:5173", want: http.StatusNoContent},
		{name: "wechat session remains public", path: "/v1/auth/wechat/sessions", want: http.StatusNoContent},
		{name: "wechat phone binding remains public", path: "/v1/auth/wechat/phone-bindings", want: http.StatusNoContent},
		{name: "other bearer mutation reaches handler", path: "/v1/resource", want: http.StatusNoContent},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, test.path, nil)
			request.Header.Set("Authorization", "Bearer arbitrary-invalid-token")
			if test.origin != "" {
				request.Header.Set("Origin", test.origin)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.want {
				t.Fatalf("status = %d, want %d", response.Code, test.want)
			}
		})
	}
}
