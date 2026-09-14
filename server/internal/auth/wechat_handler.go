package auth

import (
	"net/http"

	"github.com/GnEveLynn/FormTally/server/internal/httpapi"
)

func (h *Handler) createWeChatSession(w http.ResponseWriter, r *http.Request) {
	var request struct {
		LoginCode string `json:"loginCode"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	result, err := h.service.CreateWeChatSession(r.Context(), WeChatSessionInput{LoginCode: request.LoginCode})
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	if result.BindingRequired {
		httpapi.WriteJSON(w, http.StatusOK, struct {
			BindingRequired bool   `json:"bindingRequired"`
			BindingTicket   string `json:"bindingTicket"`
			ExpiresIn       int    `json:"expiresInSeconds"`
		}{true, result.BindingTicket, result.ExpiresInSeconds})
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, struct {
		BindingRequired bool   `json:"bindingRequired"`
		Token           string `json:"token"`
		SessionResult
	}{false, result.Token, result.Session})
}

func (h *Handler) createWeChatPhoneBinding(w http.ResponseWriter, r *http.Request) {
	var request struct {
		BindingTicket string `json:"bindingTicket"`
		PhoneCode     string `json:"phoneCode"`
		Agreements    struct {
			TermsVersion   string `json:"termsVersion"`
			PrivacyVersion string `json:"privacyVersion"`
		} `json:"agreements"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	result, token, err := h.service.BindWeChatPhone(r.Context(), WeChatPhoneBindingInput{
		BindingTicket: request.BindingTicket, PhoneCode: request.PhoneCode,
		TermsVersion: request.Agreements.TermsVersion, PrivacyVersion: request.Agreements.PrivacyVersion,
	})
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusCreated, struct {
		Token string `json:"token"`
		SessionResult
	}{token, result})
}
