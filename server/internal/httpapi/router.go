package httpapi

import (
	"log/slog"
	"net/http"
)

func NewRouter(logger *slog.Logger, allowedOrigins []string, register ...func(*http.ServeMux)) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	for _, routes := range register {
		routes(mux)
	}
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		WriteError(w, r, http.StatusNotFound, "RESOURCE_NOT_FOUND", "资源不存在")
	})
	return middleware(logger, allowedOrigins)(mux)
}
