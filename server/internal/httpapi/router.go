package httpapi

import (
	"log/slog"
	"net/http"
)

func NewRouter(logger *slog.Logger, allowedOrigins []string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, r, http.StatusNotFound, "RESOURCE_NOT_FOUND", "资源不存在")
	})
	return middleware(logger, allowedOrigins)(mux)
}
