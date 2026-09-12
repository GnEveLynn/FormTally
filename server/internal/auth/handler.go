package auth

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/GnEveLynn/FormTally/server/internal/httpapi"
)

type Handler struct {
	service      *Service
	authenticate func(*http.Request) (string, error)
}

func NewHandler(service *Service, authenticators ...func(*http.Request) (string, error)) *Handler {
	handler := &Handler{service: service}
	if len(authenticators) > 0 {
		handler.authenticate = authenticators[0]
	}
	return handler
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/auth/codes", h.requestCode)
	mux.HandleFunc("POST /v1/auth/sessions", h.createSession)
	mux.HandleFunc("GET /v1/auth/session", h.getSession)
	mux.HandleFunc("DELETE /v1/auth/session", h.deleteSession)
}

func (h *Handler) requestCode(w http.ResponseWriter, r *http.Request) {
	var input RequestCodeInput
	if !decodeJSON(w, r, &input) {
		return
	}
	userID := ""
	if input.Purpose == PurposeDeleteAccount {
		if h.authenticate == nil {
			writeServiceError(w, r, unauthenticated())
			return
		}
		var err error
		userID, err = h.authenticate(r)
		if err != nil {
			writeServiceError(w, r, unauthenticated())
			return
		}
	}
	verification, err := h.service.RequestCodeForUser(r.Context(), userID, input)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusAccepted, map[string]any{"verification": verification})
}

func (h *Handler) createSession(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Phone                 string `json:"phone"`
		Code                  string `json:"code"`
		VerificationRequestID string `json:"verificationRequestId"`
		Agreements            struct {
			TermsVersion   string `json:"termsVersion"`
			PrivacyVersion string `json:"privacyVersion"`
		} `json:"agreements"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	result, token, err := h.service.CreateSession(r.Context(), CreateSessionInput{Phone: request.Phone, Code: request.Code, VerificationRequestID: request.VerificationRequestID, TermsVersion: request.Agreements.TermsVersion, PrivacyVersion: request.Agreements.PrivacyVersion})
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: SessionCookieName, Value: token, Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode, Expires: result.Session.ExpiresAt, MaxAge: int(time.Until(result.Session.ExpiresAt).Seconds())})
	httpapi.WriteJSON(w, http.StatusCreated, result)
}

func (h *Handler) getSession(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.GetSession(r.Context(), SessionToken(r))
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) deleteSession(w http.ResponseWriter, r *http.Request) {
	if err := h.service.RevokeSession(r.Context(), SessionToken(r)); err != nil {
		writeServiceError(w, r, err)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: SessionCookieName, Value: "", Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode, Expires: time.Unix(1, 0), MaxAge: -1})
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		httpapi.WriteError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "请求格式无效")
		return false
	}
	return true
}

func writeServiceError(w http.ResponseWriter, r *http.Request, err error) {
	var apiError *Error
	if !errors.As(err, &apiError) {
		httpapi.WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "服务暂时不可用")
		return
	}
	if apiError.RetryAfter > 0 {
		w.Header().Set("Retry-After", strconv.Itoa(apiError.RetryAfter))
	}
	httpapi.WriteError(w, r, apiError.Status, apiError.Code, apiError.Message)
}
