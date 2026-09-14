package auth

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/GnEveLynn/FormTally/server/internal/wechat"
)

type fakeWeChatClient struct {
	identities map[string]wechat.LoginIdentity
	phones     map[string]wechat.Phone
	loginErr   error
	phoneErr   error
}

func (f *fakeWeChatClient) ExchangeLoginCode(_ context.Context, code string) (wechat.LoginIdentity, error) {
	if f.loginErr != nil {
		return wechat.LoginIdentity{}, f.loginErr
	}
	identity, ok := f.identities[code]
	if !ok {
		return wechat.LoginIdentity{}, &wechat.Error{Kind: wechat.ErrorInvalidCredential}
	}
	return identity, nil
}

func (f *fakeWeChatClient) ExchangePhoneCode(_ context.Context, code string) (wechat.Phone, error) {
	if f.phoneErr != nil {
		return wechat.Phone{}, f.phoneErr
	}
	phone, ok := f.phones[code]
	if !ok {
		return wechat.Phone{}, &wechat.Error{Kind: wechat.ErrorInvalidCredential}
	}
	return phone, nil
}

func TestWeChatSessionHandler(t *testing.T) {
	t.Run("rejects invalid body", func(t *testing.T) {
		service, _ := testService(t)
		router := weChatTestRouter(service, &fakeWeChatClient{})
		response := performWeChatJSON(router, http.MethodPost, "/v1/auth/wechat/sessions", `{`)
		if response.Code != http.StatusBadRequest || apiErrorCode(t, response) != "INVALID_REQUEST" {
			t.Fatalf("response = %d %s", response.Code, response.Body.String())
		}
	})

	t.Run("maps invalid login code", func(t *testing.T) {
		service, _ := testService(t)
		client := &fakeWeChatClient{loginErr: &wechat.Error{Kind: wechat.ErrorInvalidCredential}}
		response := performWeChatJSON(weChatTestRouter(service, client), http.MethodPost, "/v1/auth/wechat/sessions", `{"loginCode":"invalid"}`)
		if response.Code != http.StatusUnprocessableEntity || apiErrorCode(t, response) != "WECHAT_CODE_INVALID" {
			t.Fatalf("response = %d %s", response.Code, response.Body.String())
		}
	})

	t.Run("new identity receives ticket without creating account", func(t *testing.T) {
		service, _ := testService(t)
		client := &fakeWeChatClient{identities: map[string]wechat.LoginIdentity{"login-code": {OpenID: "openid-new", SessionKey: "must-not-persist"}}}
		response := performWeChatJSON(weChatTestRouter(service, client), http.MethodPost, "/v1/auth/wechat/sessions", `{"loginCode":"login-code"}`)
		if response.Code != http.StatusOK {
			t.Fatalf("response = %d %s", response.Code, response.Body.String())
		}
		var body struct {
			BindingRequired bool   `json:"bindingRequired"`
			BindingTicket   string `json:"bindingTicket"`
			ExpiresIn       int    `json:"expiresInSeconds"`
			Token           string `json:"token"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if !body.BindingRequired || body.BindingTicket == "" || body.ExpiresIn != 300 || body.Token != "" {
			t.Fatalf("body = %+v", body)
		}
		var users, sessions, tickets, leakedSessionKeys int
		if err := service.store.pool.QueryRow(context.Background(), `select (select count(*) from users),(select count(*) from sessions),(select count(*) from wechat_binding_tickets),(select count(*) from wechat_binding_tickets where openid='must-not-persist' or union_id='must-not-persist')`).Scan(&users, &sessions, &tickets, &leakedSessionKeys); err != nil {
			t.Fatal(err)
		}
		if users != 0 || sessions != 0 || tickets != 1 || leakedSessionKeys != 0 {
			t.Fatalf("users = %d, sessions = %d, tickets = %d, leaked session keys = %d", users, sessions, tickets, leakedSessionKeys)
		}
	})

	t.Run("existing identity receives session and token", func(t *testing.T) {
		service, _ := testService(t)
		now := time.Date(2026, 9, 14, 8, 0, 0, 0, time.UTC)
		service.now = func() time.Time { return now }
		seed := bindWeChatForTest(t, service.store, "seed-ticket", "openid-existing", "+8613812345678", now)
		client := &fakeWeChatClient{identities: map[string]wechat.LoginIdentity{"login-code": {OpenID: "openid-existing", SessionKey: "session-key"}}}
		response := performWeChatJSON(weChatTestRouter(service, client), http.MethodPost, "/v1/auth/wechat/sessions", `{"loginCode":"login-code"}`)
		if response.Code != http.StatusOK || len(response.Result().Cookies()) != 0 {
			t.Fatalf("response = %d, cookies = %+v, body = %s", response.Code, response.Result().Cookies(), response.Body.String())
		}
		var body struct {
			BindingRequired bool         `json:"bindingRequired"`
			Token           string       `json:"token"`
			User            UserView     `json:"user"`
			Session         SessionView  `json:"session"`
			Consents        ConsentsView `json:"consents"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body.BindingRequired || body.Token == "" || body.User.ID != seed.User.ID || strings.Count(response.Body.String(), `"token"`) != 1 {
			t.Fatalf("body = %s", response.Body.String())
		}
		current, err := service.GetSession(context.Background(), body.Token)
		if err != nil || current.User.ID != seed.User.ID {
			t.Fatalf("current = %+v, err = %v", current, err)
		}
		var clientType string
		if err := service.store.pool.QueryRow(context.Background(), `select client_type from sessions where token_hash=$1`, tokenHash(body.Token)).Scan(&clientType); err != nil || clientType != string(ClientWeChatMiniProgram) {
			t.Fatalf("client_type = %q, err = %v", clientType, err)
		}
	})
}

func TestWeChatSessionHandlerMapsProviderAvailabilityErrors(t *testing.T) {
	for _, test := range []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{name: "rate limited", err: &wechat.Error{Kind: wechat.ErrorRateLimited}, status: http.StatusTooManyRequests, code: "WECHAT_RATE_LIMITED"},
		{name: "unavailable", err: &wechat.Error{Kind: wechat.ErrorUnavailable}, status: http.StatusServiceUnavailable, code: "WECHAT_UNAVAILABLE"},
		{name: "invalid response", err: &wechat.Error{Kind: wechat.ErrorInvalidResponse}, status: http.StatusServiceUnavailable, code: "WECHAT_UNAVAILABLE"},
	} {
		t.Run(test.name, func(t *testing.T) {
			service, _ := testService(t)
			client := &fakeWeChatClient{loginErr: test.err}
			response := performWeChatJSON(weChatTestRouter(service, client), http.MethodPost, "/v1/auth/wechat/sessions", `{"loginCode":"login-code"}`)
			if response.Code != test.status || apiErrorCode(t, response) != test.code {
				t.Fatalf("response = %d %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestWeChatPhoneBindingHandlerErrors(t *testing.T) {
	t.Run("missing ticket", func(t *testing.T) {
		service, _ := testService(t)
		client := &fakeWeChatClient{phones: map[string]wechat.Phone{"phone-code": {Number: "+8613812345678"}}}
		response := performWeChatJSON(weChatTestRouter(service, client), http.MethodPost, "/v1/auth/wechat/phone-bindings", validWeChatBindingBody("missing", "phone-code"))
		if response.Code != http.StatusUnprocessableEntity || apiErrorCode(t, response) != "WECHAT_BINDING_TICKET_INVALID" {
			t.Fatalf("response = %d %s", response.Code, response.Body.String())
		}
	})

	t.Run("expired ticket", func(t *testing.T) {
		service, _ := testService(t)
		now := time.Date(2026, 9, 14, 8, 0, 0, 0, time.UTC)
		service.now = func() time.Time { return now }
		raw := "expired-ticket"
		hash := sha256.Sum256([]byte(raw))
		if err := service.store.CreateWeChatBindingTicket(context.Background(), hash[:], "openid-expired", "", now.Add(-10*time.Minute), now.Add(-5*time.Minute)); err != nil {
			t.Fatal(err)
		}
		client := &fakeWeChatClient{phones: map[string]wechat.Phone{"phone-code": {Number: "+8613812345678"}}}
		response := performWeChatJSON(weChatTestRouter(service, client), http.MethodPost, "/v1/auth/wechat/phone-bindings", validWeChatBindingBody(raw, "phone-code"))
		if response.Code != http.StatusGone || apiErrorCode(t, response) != "WECHAT_BINDING_TICKET_EXPIRED" {
			t.Fatalf("response = %d %s", response.Code, response.Body.String())
		}
	})

	t.Run("consumed ticket", func(t *testing.T) {
		service, _ := testService(t)
		now := time.Date(2026, 9, 14, 8, 0, 0, 0, time.UTC)
		service.now = func() time.Time { return now }
		raw := "consumed-ticket"
		bindWeChatForTest(t, service.store, raw, "openid-consumed", "+8613812345678", now)
		client := &fakeWeChatClient{phones: map[string]wechat.Phone{"phone-code": {Number: "+8613812345678"}}}
		response := performWeChatJSON(weChatTestRouter(service, client), http.MethodPost, "/v1/auth/wechat/phone-bindings", validWeChatBindingBody(raw, "phone-code"))
		if response.Code != http.StatusConflict || apiErrorCode(t, response) != "WECHAT_BINDING_TICKET_CONSUMED" {
			t.Fatalf("response = %d %s", response.Code, response.Body.String())
		}
	})

	t.Run("invalid phone code", func(t *testing.T) {
		service, _ := testService(t)
		client := &fakeWeChatClient{phoneErr: &wechat.Error{Kind: wechat.ErrorInvalidCredential}}
		response := performWeChatJSON(weChatTestRouter(service, client), http.MethodPost, "/v1/auth/wechat/phone-bindings", validWeChatBindingBody("ticket", "bad-phone-code"))
		if response.Code != http.StatusUnprocessableEntity || apiErrorCode(t, response) != "WECHAT_PHONE_UNAVAILABLE" {
			t.Fatalf("response = %d %s", response.Code, response.Body.String())
		}
	})

	t.Run("outdated agreements", func(t *testing.T) {
		service, _ := testService(t)
		client := &fakeWeChatClient{phones: map[string]wechat.Phone{"phone-code": {Number: "+8613812345678"}}}
		body := `{"bindingTicket":"ticket","phoneCode":"phone-code","agreements":{"termsVersion":"old","privacyVersion":"old"}}`
		response := performWeChatJSON(weChatTestRouter(service, client), http.MethodPost, "/v1/auth/wechat/phone-bindings", body)
		if response.Code != http.StatusConflict || apiErrorCode(t, response) != "AGREEMENT_VERSION_OUTDATED" {
			t.Fatalf("response = %d %s", response.Code, response.Body.String())
		}
	})

	t.Run("identity conflict", func(t *testing.T) {
		service, _ := testService(t)
		now := time.Date(2026, 9, 14, 8, 0, 0, 0, time.UTC)
		service.now = func() time.Time { return now }
		bindWeChatForTest(t, service.store, "owner-ticket", "openid-conflict-handler", "+8613812345678", now)
		if _, err := service.store.pool.Exec(context.Background(), `insert into users(id,phone) values ('other-handler-user','+8613912345678')`); err != nil {
			t.Fatal(err)
		}
		raw := "conflict-ticket"
		hash := sha256.Sum256([]byte(raw))
		if err := service.store.CreateWeChatBindingTicket(context.Background(), hash[:], "openid-conflict-handler", "", now, now.Add(5*time.Minute)); err != nil {
			t.Fatal(err)
		}
		client := &fakeWeChatClient{phones: map[string]wechat.Phone{"phone-code": {Number: "+8613912345678"}}}
		response := performWeChatJSON(weChatTestRouter(service, client), http.MethodPost, "/v1/auth/wechat/phone-bindings", validWeChatBindingBody(raw, "phone-code"))
		if response.Code != http.StatusConflict || apiErrorCode(t, response) != "IDENTITY_CONFLICT" {
			t.Fatalf("response = %d %s", response.Code, response.Body.String())
		}
	})
}

func TestWeChatPhoneBindingCreatesOrLinksAccount(t *testing.T) {
	for _, test := range []struct {
		name       string
		seedPhone  bool
		wantUserID string
	}{
		{name: "links existing phone", seedPhone: true, wantUserID: "existing-phone-user"},
		{name: "creates new phone user"},
	} {
		t.Run(test.name, func(t *testing.T) {
			service, _ := testService(t)
			now := time.Date(2026, 9, 14, 8, 0, 0, 0, time.UTC)
			service.now = func() time.Time { return now }
			if test.seedPhone {
				if _, err := service.store.pool.Exec(context.Background(), `insert into users(id,phone) values ('existing-phone-user','+8613812345678')`); err != nil {
					t.Fatal(err)
				}
			}
			client := &fakeWeChatClient{
				identities: map[string]wechat.LoginIdentity{"login-code": {OpenID: "openid-success", UnionID: "union-success", SessionKey: "session-key"}},
				phones:     map[string]wechat.Phone{"phone-code": {Number: "+8613812345678"}},
			}
			router := weChatTestRouter(service, client)
			sessionResponse := performWeChatJSON(router, http.MethodPost, "/v1/auth/wechat/sessions", `{"loginCode":"login-code"}`)
			var sessionBody struct {
				BindingTicket string `json:"bindingTicket"`
			}
			if err := json.Unmarshal(sessionResponse.Body.Bytes(), &sessionBody); err != nil || sessionBody.BindingTicket == "" {
				t.Fatalf("session response = %d %s, err = %v", sessionResponse.Code, sessionResponse.Body.String(), err)
			}
			response := performWeChatJSON(router, http.MethodPost, "/v1/auth/wechat/phone-bindings", validWeChatBindingBody(sessionBody.BindingTicket, "phone-code"))
			if response.Code != http.StatusCreated || len(response.Result().Cookies()) != 0 || strings.Count(response.Body.String(), `"token"`) != 1 {
				t.Fatalf("response = %d, cookies = %+v, body = %s", response.Code, response.Result().Cookies(), response.Body.String())
			}
			var body struct {
				Token string   `json:"token"`
				User  UserView `json:"user"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body.Token == "" {
				t.Fatalf("body = %s, err = %v", response.Body.String(), err)
			}
			if test.wantUserID != "" && body.User.ID != test.wantUserID {
				t.Fatalf("user ID = %q", body.User.ID)
			}
			var consents int
			var clientType string
			if err := service.store.pool.QueryRow(context.Background(), `select count(*) from user_consents where user_id=$1 and kind in ('terms','privacy')`, body.User.ID).Scan(&consents); err != nil {
				t.Fatal(err)
			}
			if err := service.store.pool.QueryRow(context.Background(), `select client_type from sessions where token_hash=$1`, tokenHash(body.Token)).Scan(&clientType); err != nil {
				t.Fatal(err)
			}
			if consents != 2 || clientType != string(ClientWeChatMiniProgram) {
				t.Fatalf("consents = %d, client_type = %q", consents, clientType)
			}
		})
	}
}

func weChatTestRouter(service *Service, client wechat.Client) http.Handler {
	configured := NewService(service.store, service.sender, client)
	configured.now = service.now
	*service = *configured
	mux := http.NewServeMux()
	NewHandler(service).Register(mux)
	return mux
}

func performWeChatJSON(handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func validWeChatBindingBody(ticket, phoneCode string) string {
	body, _ := json.Marshal(map[string]any{
		"bindingTicket": ticket,
		"phoneCode":     phoneCode,
		"agreements": map[string]string{
			"termsVersion":   CurrentTermsVersion,
			"privacyVersion": CurrentPrivacyVersion,
		},
	})
	return string(body)
}

func apiErrorCode(t *testing.T, response *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body.Error.Code
}
