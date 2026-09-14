package analysis

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/GnEveLynn/FormTally/server/internal/httpapi"
	"github.com/GnEveLynn/FormTally/server/internal/idempotency"
	"github.com/GnEveLynn/FormTally/server/internal/storage"
)

type Handler struct {
	service      *Service
	authenticate func(*http.Request) (string, error)
	logger       *slog.Logger
}

func NewHandler(service *Service, authenticate func(*http.Request) (string, error), loggers ...*slog.Logger) *Handler {
	var logger *slog.Logger
	if len(loggers) > 0 {
		logger = loggers[0]
	}
	return &Handler{service: service, authenticate: authenticate, logger: logger}
}
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/meal-analyses", h.create)
	mux.HandleFunc("GET /v1/meal-analyses/{analysisId}", h.get)
	mux.HandleFunc("POST /v1/meal-analyses/{analysisId}/retry", h.retry)
	mux.HandleFunc("DELETE /v1/meal-analyses/{analysisId}", h.discard)
}
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.user(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 12<<20)
	if err := r.ParseMultipartForm(12 << 20); err != nil {
		httpapi.WriteError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "上传格式无效")
		return
	}
	file, _, err := r.FormFile("image")
	if err != nil {
		httpapi.WriteError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "请选择图片")
		return
	}
	defer file.Close()
	processed, err := storage.ProcessImage(file, 10<<20, 40_000_000)
	if err != nil {
		h.imageError(w, r, err)
		return
	}
	view, err := h.service.Create(r.Context(), userID, r.Header.Get("Idempotency-Key"), CreateInput{ProcessingMode: r.FormValue("processingMode"), AIConsentVersion: r.FormValue("aiConsentVersion"), OccurredAt: r.FormValue("occurredAt"), MealType: r.FormValue("mealType"), Image: processed.Bytes, Width: processed.Width, Height: processed.Height})
	if h.writeServiceError(w, r, err) {
		return
	}
	logAnalysisResult(h.logger, httpapi.RequestID(r.Context()), view)
	httpapi.WriteJSON(w, http.StatusCreated, map[string]any{"analysis": view})
}
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.user(w, r)
	if !ok {
		return
	}
	view, err := h.service.Get(r.Context(), userID, r.PathValue("analysisId"))
	if err == nil && view.ID == "" {
		httpapi.WriteError(w, r, http.StatusNotFound, "RESOURCE_NOT_FOUND", "资源不存在")
		return
	}
	if h.writeServiceError(w, r, err) {
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"analysis": view})
}

func logAnalysisResult(logger *slog.Logger, requestID string, view View) {
	if logger == nil {
		return
	}
	failureCode := ""
	if view.Failure != nil {
		failureCode = view.Failure.Code
	}
	logger.Info("meal analysis completed",
		"request_id", requestID,
		"analysis_id", view.ID,
		"status", view.Status,
		"processing_mode", view.ProcessingMode,
		"item_count", len(view.Items),
		"failure_code", failureCode,
		"model", view.Metadata.Model,
		"prompt_version", view.Metadata.PromptVersion,
		"ai_response_status", view.Metadata.ResponseStatus,
		"ai_duration_ms", view.Metadata.Duration.Milliseconds(),
	)
}
func (h *Handler) retry(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.user(w, r)
	if !ok {
		return
	}
	var input struct {
		AIConsentVersion string `json:"aiConsentVersion"`
		ExpectedRevision int    `json:"expectedRevision"`
	}
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&input) != nil {
		httpapi.WriteError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "请求格式无效")
		return
	}
	view, err := h.service.Retry(r.Context(), userID, r.PathValue("analysisId"), r.Header.Get("Idempotency-Key"), input.AIConsentVersion, input.ExpectedRevision)
	if h.writeServiceError(w, r, err) {
		return
	}
	logAnalysisResult(h.logger, httpapi.RequestID(r.Context()), view)
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"analysis": view})
}
func (h *Handler) discard(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.user(w, r)
	if !ok {
		return
	}
	revision, err := strconv.Atoi(r.URL.Query().Get("expectedRevision"))
	if err != nil {
		httpapi.WriteError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "expectedRevision 无效")
		return
	}
	if h.writeServiceError(w, r, h.service.Discard(r.Context(), userID, r.PathValue("analysisId"), revision)) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (h *Handler) user(w http.ResponseWriter, r *http.Request) (string, bool) {
	id, err := h.authenticate(r)
	if err != nil {
		httpapi.WriteError(w, r, http.StatusUnauthorized, "UNAUTHENTICATED", "请先登录")
		return "", false
	}
	return id, true
}
func (h *Handler) imageError(w http.ResponseWriter, r *http.Request, err error) {
	for _, candidate := range []struct {
		code    string
		status  int
		message string
	}{{"IMAGE_TOO_LARGE", 413, "图片不能超过 10 MB"}, {"IMAGE_DIMENSIONS_TOO_LARGE", 413, "图片像素尺寸过大"}, {"IMAGE_UNSUPPORTED", 415, "暂不支持这种图片格式"}} {
		if storage.IsImageError(err, candidate.code) {
			httpapi.WriteError(w, r, candidate.status, candidate.code, candidate.message)
			return
		}
	}
	httpapi.WriteError(w, r, 500, "INTERNAL_ERROR", "图片处理失败")
}
func (h *Handler) writeServiceError(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, ErrAIConsentRequired):
		httpapi.WriteError(w, r, 422, "AI_CONSENT_REQUIRED", "请先确认 AI 图片处理说明")
	case errors.Is(err, ErrExpired):
		httpapi.WriteError(w, r, 410, "ANALYSIS_EXPIRED", "分析草稿已过期")
	case errors.Is(err, ErrStateConflict):
		httpapi.WriteError(w, r, 409, "ANALYSIS_STATE_CONFLICT", "当前草稿状态不允许此操作")
	case errors.Is(err, ErrRevisionConflict):
		httpapi.WriteError(w, r, 409, "REVISION_CONFLICT", "草稿已更新，请刷新后重试")
	case errors.Is(err, idempotency.ErrConflict):
		httpapi.WriteError(w, r, 409, "IDEMPOTENCY_CONFLICT", "幂等键对应了不同请求")
	case errors.Is(err, idempotency.ErrInProgress):
		httpapi.WriteError(w, r, 409, "OPERATION_IN_PROGRESS", "相同操作仍在处理中")
	case errors.Is(err, ErrInvalid):
		httpapi.WriteError(w, r, 422, "VALIDATION_FAILED", "分析参数无效")
	default:
		httpapi.WriteError(w, r, 500, "INTERNAL_ERROR", "服务暂时不可用")
	}
	return true
}
