package httpapi

import (
	"context"
	"crypto/rand"
	"log/slog"
	"net/http"
	"time"
)

type bearerCredentialKey struct{}

func WithBearerCredential(ctx context.Context) context.Context {
	return context.WithValue(ctx, bearerCredentialKey{}, true)
}

func hasBearerCredential(ctx context.Context) bool {
	value, _ := ctx.Value(bearerCredentialKey{}).(bool)
	return value
}

func middleware(logger *slog.Logger, allowedOrigins []string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		allowed[origin] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return withRequestID(withAccessLog(logger, withRecovery(withOrigin(allowed, next))))
	}
}

func withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if !validRequestID(id) {
			id = "req_" + rand.Text()
		}
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey{}, id)))
	})
}

func validRequestID(value string) bool {
	if len(value) < 8 || len(value) > 128 {
		return false
	}
	for _, char := range value {
		if !(char >= 'a' && char <= 'z') && !(char >= 'A' && char <= 'Z') &&
			!(char >= '0' && char <= '9') && char != '-' && char != '_' && char != '.' && char != ':' {
			return false
		}
	}
	return true
}

func withRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recover() != nil {
				WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "服务暂时不可用")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func withOrigin(allowed map[string]struct{}, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if changesState(r.Method) && !publicMutation(r) && (browserAuthMutation(r) || !hasBearerCredential(r.Context())) {
			if _, ok := allowed[r.Header.Get("Origin")]; !ok {
				WriteError(w, r, http.StatusForbidden, "ORIGIN_NOT_ALLOWED", "请求来源不被允许")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func browserAuthMutation(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	return r.URL.Path == "/v1/auth/codes" || r.URL.Path == "/v1/auth/sessions"
}

func publicMutation(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	return r.URL.Path == "/v1/auth/wechat/sessions" || r.URL.Path == "/v1/auth/wechat/phone-bindings"
}

func changesState(method string) bool {
	return method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch || method == http.MethodDelete
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(body)
}

func withAccessLog(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		response := &statusWriter{ResponseWriter: w}
		next.ServeHTTP(response, r)
		status := response.status
		if status == 0 {
			status = http.StatusOK
		}
		logger.Info("http request",
			"request_id", RequestID(r.Context()),
			"method", r.Method,
			"path", r.URL.Path,
			"status", status,
			"duration_ms", time.Since(started).Milliseconds(),
		)
	})
}
