package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRouterAppliesJSONProtocol(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := NewRouter(logger, []string{"http://127.0.0.1:5173"})

	t.Run("success", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		request.Header.Set("X-Request-ID", "client-request-123456")
		response := httptest.NewRecorder()

		handler.ServeHTTP(response, request)

		assertProtocolHeaders(t, response, "client-request-123456")
		if response.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
		}
	})

	t.Run("unknown route", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/missing", nil)
		response := httptest.NewRecorder()

		handler.ServeHTTP(response, request)

		requestID := response.Header().Get("X-Request-ID")
		assertProtocolHeaders(t, response, requestID)
		if response.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
		}
		var body errorResponse
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body.Error.Code != "RESOURCE_NOT_FOUND" || body.Error.RequestID != requestID {
			t.Fatalf("error = %+v", body.Error)
		}
	})
}

func TestOriginMiddleware(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		WriteJSON(w, http.StatusCreated, map[string]string{"status": "created"})
	})
	handler := middleware(logger, []string{"http://127.0.0.1:5173"})(next)

	for _, test := range []struct {
		name   string
		origin string
		status int
	}{
		{name: "allowed", origin: "http://127.0.0.1:5173", status: http.StatusCreated},
		{name: "missing", status: http.StatusForbidden},
		{name: "not allowed", origin: "https://evil.example", status: http.StatusForbidden},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/v1/resource", nil)
			request.Header.Set("Origin", test.origin)
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if response.Code != test.status {
				t.Fatalf("status = %d, want %d", response.Code, test.status)
			}
			if test.status == http.StatusForbidden {
				var body errorResponse
				if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if body.Error.Code != "ORIGIN_NOT_ALLOWED" {
					t.Fatalf("code = %q", body.Error.Code)
				}
			}
		})
	}
}

func TestOriginMiddlewareUsesCredentialKindAndExactPublicRoutes(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		WriteJSON(w, http.StatusCreated, map[string]string{"status": "created"})
	})
	handler := middleware(logger, []string{"http://127.0.0.1:5173"})(next)

	tests := []struct {
		name      string
		path      string
		configure func(*http.Request) *http.Request
		want      int
	}{
		{
			name: "cookie write still needs origin",
			path: "/v1/resource",
			configure: func(request *http.Request) *http.Request {
				request.AddCookie(&http.Cookie{Name: "formtally_session", Value: "cookie-token"})
				return request
			},
			want: http.StatusForbidden,
		},
		{
			name: "bearer write does not need origin",
			path: "/v1/resource",
			configure: func(request *http.Request) *http.Request {
				return request.WithContext(WithBearerCredential(request.Context()))
			},
			want: http.StatusCreated,
		},
		{
			name:      "wechat session is public",
			path:      "/v1/auth/wechat/sessions",
			configure: func(request *http.Request) *http.Request { return request },
			want:      http.StatusCreated,
		},
		{
			name:      "wechat phone binding is public",
			path:      "/v1/auth/wechat/phone-bindings",
			configure: func(request *http.Request) *http.Request { return request },
			want:      http.StatusCreated,
		},
		{
			name:      "wechat route prefix is not public",
			path:      "/v1/auth/wechat/sessions/extra",
			configure: func(request *http.Request) *http.Request { return request },
			want:      http.StatusForbidden,
		},
		{
			name:      "other anonymous write is not public",
			path:      "/v1/resource",
			configure: func(request *http.Request) *http.Request { return request },
			want:      http.StatusForbidden,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := test.configure(httptest.NewRequest(http.MethodPost, test.path, nil))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.want {
				t.Fatalf("status = %d, want %d", response.Code, test.want)
			}
		})
	}
}

func TestMiddlewareRecoversAndLogsOnlySafeFields(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	handler := middleware(logger, nil)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("verification-code-123456")
	}))
	request := httptest.NewRequest(http.MethodGet, "/panic?token=query-secret", strings.NewReader("image-body-secret"))
	request.Header.Set("Authorization", "Bearer auth-secret")
	request.Header.Set("Cookie", "session=cookie-secret")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	var body errorResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Error.Code != "INTERNAL_ERROR" || body.Error.RequestID != response.Header().Get("X-Request-ID") {
		t.Fatalf("error = %+v", body.Error)
	}
	logText := logs.String()
	for _, secret := range []string{"verification-code-123456", "query-secret", "image-body-secret", "auth-secret", "cookie-secret"} {
		if strings.Contains(logText, secret) {
			t.Fatalf("logs contain secret %q: %s", secret, logText)
		}
	}
	for _, field := range []string{"request_id", "method", "path", "status", "duration_ms"} {
		if !strings.Contains(logText, `"`+field+`"`) {
			t.Fatalf("logs missing field %q: %s", field, logText)
		}
	}
}

func assertProtocolHeaders(t *testing.T, response *httptest.ResponseRecorder, requestID string) {
	t.Helper()
	if requestID == "" {
		t.Fatal("X-Request-ID is empty")
	}
	if got := response.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
	if got := response.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q", got)
	}
}
