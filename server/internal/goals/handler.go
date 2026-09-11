package goals

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
	mux.HandleFunc("GET /v1/goals", h.get)
	mux.HandleFunc("POST /v1/goal-previews", h.preview)
	mux.HandleFunc("PUT /v1/goals", h.save)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}
	view, err := h.service.Get(r.Context(), userID)
	if err != nil {
		httpapi.WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "服务暂时不可用")
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, view)
}

func (h *Handler) preview(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}
	input, ok := decodeSettings(w, r)
	if !ok {
		return
	}
	preview, err := h.service.Preview(r.Context(), userID, input)
	if h.writeError(w, r, err) {
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"preview": preview})
}

func (h *Handler) save(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}
	input, ok := decodeSettings(w, r)
	if !ok {
		return
	}
	result, err := h.service.Save(r.Context(), userID, input)
	if h.writeError(w, r, err) {
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, result)
}

func decodeSettings(w http.ResponseWriter, r *http.Request) (SettingsInput, bool) {
	var input SettingsInput
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		httpapi.WriteError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "请求格式无效")
		return SettingsInput{}, false
	}
	return input, true
}

func (h *Handler) userID(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID, err := h.authenticate(r)
	if err != nil {
		httpapi.WriteError(w, r, http.StatusUnauthorized, "UNAUTHENTICATED", "请先登录")
		return "", false
	}
	return userID, true
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, ErrRevisionConflict):
		httpapi.WriteError(w, r, http.StatusConflict, "REVISION_CONFLICT", "目标设置已更新，请刷新后重试")
	case errors.Is(err, ErrAutomaticGoalNotEligible):
		httpapi.WriteError(w, r, http.StatusUnprocessableEntity, "AUTO_GOAL_NOT_ELIGIBLE", "当前资料不允许自动计算")
	case errors.Is(err, ErrInvalidGoal):
		httpapi.WriteError(w, r, http.StatusUnprocessableEntity, "VALIDATION_FAILED", "目标设置不符合允许范围")
	case errors.Is(err, ErrProfileRequired):
		httpapi.WriteError(w, r, http.StatusUnprocessableEntity, "AUTO_GOAL_NOT_ELIGIBLE", "请先填写身体资料")
	default:
		httpapi.WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "服务暂时不可用")
	}
	return true
}
