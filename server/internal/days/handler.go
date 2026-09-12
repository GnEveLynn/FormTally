package days

import (
	"errors"
	"github.com/GnEveLynn/FormTally/server/internal/httpapi"
	"net/http"
)

type Handler struct {
	service      *Service
	authenticate func(*http.Request) (string, error)
}

func NewHandler(service *Service, authenticate func(*http.Request) (string, error)) *Handler {
	return &Handler{service: service, authenticate: authenticate}
}
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/days/{localDate}", h.get)
	mux.HandleFunc("GET /v1/history", h.history)
}
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	user, ok := h.user(w, r)
	if !ok {
		return
	}
	view, err := h.service.Get(r.Context(), user, r.PathValue("localDate"))
	if errors.Is(err, ErrInvalidDate) {
		httpapi.WriteError(w, r, 422, "VALIDATION_FAILED", "日期无效或位于未来")
		return
	}
	if err != nil {
		httpapi.WriteError(w, r, 500, "INTERNAL_ERROR", "服务暂时不可用")
		return
	}
	httpapi.WriteJSON(w, 200, map[string]any{"day": view})
}
func (h *Handler) history(w http.ResponseWriter, r *http.Request) {
	user, ok := h.user(w, r)
	if !ok {
		return
	}
	month := h.service.ResolveMonth(r.URL.Query().Get("month"))
	view, err := h.service.History(r.Context(), user, month)
	if errors.Is(err, ErrInvalidDate) {
		httpapi.WriteError(w, r, 400, "INVALID_REQUEST", "月份无效")
		return
	}
	if err != nil {
		httpapi.WriteError(w, r, 500, "INTERNAL_ERROR", "服务暂时不可用")
		return
	}
	httpapi.WriteJSON(w, 200, view)
}
func (h *Handler) user(w http.ResponseWriter, r *http.Request) (string, bool) {
	user, err := h.authenticate(r)
	if err != nil {
		httpapi.WriteError(w, r, 401, "UNAUTHENTICATED", "请先登录")
		return "", false
	}
	return user, true
}
