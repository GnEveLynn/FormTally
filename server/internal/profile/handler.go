package profile

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/GnEveLynn/FormTally/server/internal/httpapi"
)

type Handler struct {
	service      *Service
	authenticate func(*http.Request) (string, error)
}

func NewHandler(service *Service, authenticate func(*http.Request) (string, error)) *Handler {
	return &Handler{service: service, authenticate: authenticate}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/profile", h.get)
	mux.HandleFunc("PUT /v1/profile", h.put)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}
	profile, err := h.service.Get(r.Context(), userID)
	if err != nil {
		httpapi.WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "服务暂时不可用")
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"profile": profile})
}

func (h *Handler) put(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}
	var input PutInput
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		httpapi.WriteError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "请求格式无效")
		return
	}
	profile, err := h.service.Put(r.Context(), userID, input)
	if errors.Is(err, ErrValidation) {
		httpapi.WriteError(w, r, http.StatusUnprocessableEntity, "VALIDATION_FAILED", "身体资料不符合允许范围")
		return
	}
	if errors.Is(err, ErrRevisionConflict) {
		httpapi.WriteError(w, r, http.StatusConflict, "REVISION_CONFLICT", "身体资料已更新，请刷新后重试")
		return
	}
	if err != nil {
		httpapi.WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "服务暂时不可用")
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"profile": profile})
}

func (h *Handler) userID(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID, err := h.authenticate(r)
	if err != nil {
		httpapi.WriteError(w, r, http.StatusUnauthorized, "UNAUTHENTICATED", "请先登录")
		return "", false
	}
	return userID, true
}
