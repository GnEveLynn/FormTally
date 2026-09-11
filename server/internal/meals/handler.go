package meals

import (
	"encoding/json"
	"errors"
	"github.com/GnEveLynn/FormTally/server/internal/httpapi"
	"github.com/GnEveLynn/FormTally/server/internal/idempotency"
	"io"
	"net/http"
	"strconv"
)

type Handler struct {
	service      *Service
	authenticate func(*http.Request) (string, error)
}

func NewHandler(service *Service, authenticate func(*http.Request) (string, error)) *Handler {
	return &Handler{service: service, authenticate: authenticate}
}
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/meals", h.create)
	mux.HandleFunc("GET /v1/meals/{mealId}", h.get)
	mux.HandleFunc("PATCH /v1/meals/{mealId}", h.update)
	mux.HandleFunc("DELETE /v1/meals/{mealId}/image", h.removeImage)
	mux.HandleFunc("DELETE /v1/meals/{mealId}", h.delete)
}
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	user, ok := h.user(w, r)
	if !ok {
		return
	}
	var input CreateInput
	if !decode(r, &input) {
		httpapi.WriteError(w, r, 400, "INVALID_REQUEST", "请求格式无效")
		return
	}
	result, err := h.service.Create(r.Context(), user, r.Header.Get("Idempotency-Key"), input)
	if h.writeError(w, r, err) {
		return
	}
	httpapi.WriteJSON(w, 201, result)
}
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	user, ok := h.user(w, r)
	if !ok {
		return
	}
	meal, err := h.service.Get(r.Context(), user, r.PathValue("mealId"))
	if h.writeError(w, r, err) {
		return
	}
	httpapi.WriteJSON(w, 200, map[string]any{"meal": meal})
}
func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	user, ok := h.user(w, r)
	if !ok {
		return
	}
	var input UpdateInput
	if !decode(r, &input) {
		httpapi.WriteError(w, r, 400, "INVALID_REQUEST", "请求格式无效")
		return
	}
	result, err := h.service.Update(r.Context(), user, r.PathValue("mealId"), input)
	if h.writeError(w, r, err) {
		return
	}
	httpapi.WriteJSON(w, 200, result)
}
func (h *Handler) removeImage(w http.ResponseWriter, r *http.Request) {
	user, ok := h.user(w, r)
	if !ok {
		return
	}
	revision, err := strconv.Atoi(r.URL.Query().Get("expectedRevision"))
	if err != nil {
		httpapi.WriteError(w, r, 400, "INVALID_REQUEST", "expectedRevision 无效")
		return
	}
	meal, err := h.service.RemoveImage(r.Context(), user, r.PathValue("mealId"), revision)
	if h.writeError(w, r, err) {
		return
	}
	httpapi.WriteJSON(w, 200, map[string]any{"meal": meal})
}
func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	user, ok := h.user(w, r)
	if !ok {
		return
	}
	revision, err := strconv.Atoi(r.URL.Query().Get("expectedRevision"))
	if err != nil {
		httpapi.WriteError(w, r, 400, "INVALID_REQUEST", "expectedRevision 无效")
		return
	}
	result, err := h.service.Delete(r.Context(), user, r.PathValue("mealId"), revision)
	if h.writeError(w, r, err) {
		return
	}
	httpapi.WriteJSON(w, 200, result)
}
func (h *Handler) user(w http.ResponseWriter, r *http.Request) (string, bool) {
	user, err := h.authenticate(r)
	if err != nil {
		httpapi.WriteError(w, r, 401, "UNAUTHENTICATED", "请先登录")
		return "", false
	}
	return user, true
}
func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, ErrValidation):
		httpapi.WriteError(w, r, 422, "VALIDATION_FAILED", "部分内容需要修改")
	case errors.Is(err, ErrNotFound):
		httpapi.WriteError(w, r, 404, "RESOURCE_NOT_FOUND", "资源不存在")
	case errors.Is(err, ErrRevisionConflict):
		httpapi.WriteError(w, r, 409, "REVISION_CONFLICT", "记录已更新，请刷新后重试")
	case errors.Is(err, ErrAnalysisExpired):
		httpapi.WriteError(w, r, 410, "ANALYSIS_EXPIRED", "分析草稿已过期")
	case errors.Is(err, ErrAnalysisState):
		httpapi.WriteError(w, r, 409, "ANALYSIS_STATE_CONFLICT", "当前草稿状态不允许保存")
	case errors.Is(err, idempotency.ErrConflict):
		httpapi.WriteError(w, r, 409, "IDEMPOTENCY_CONFLICT", "幂等键对应了不同请求")
	case errors.Is(err, idempotency.ErrInProgress):
		httpapi.WriteError(w, r, 409, "OPERATION_IN_PROGRESS", "相同操作仍在处理中")
	default:
		httpapi.WriteError(w, r, 500, "INTERNAL_ERROR", "服务暂时不可用")
	}
	return true
}
func decode(r *http.Request, value any) bool {
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if decoder.Decode(value) != nil {
		return false
	}
	return decoder.Decode(new(any)) == io.EOF
}
