package account

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/GnEveLynn/FormTally/server/internal/auth"
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
	mux.HandleFunc("POST /v1/account-deletions", h.delete)
}
func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	userID, err := h.authenticate(r)
	if err != nil {
		httpapi.WriteError(w, r, http.StatusUnauthorized, "UNAUTHENTICATED", "请先登录")
		return
	}
	var input DeleteInput
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&input) != nil || decoder.Decode(new(any)) != io.EOF {
		httpapi.WriteError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "请求格式无效")
		return
	}
	deletion, err := h.service.Delete(r.Context(), userID, input)
	if err != nil {
		code, name, message := status(err)
		httpapi.WriteError(w, r, code, name, message)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: auth.SessionCookieName, Value: "", Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode, Expires: time.Unix(1, 0), MaxAge: -1})
	httpapi.WriteJSON(w, http.StatusAccepted, map[string]any{"accountDeletion": deletion})
}
