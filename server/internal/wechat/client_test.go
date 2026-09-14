package wechat

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestClientExchangeLoginCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sns/jscode2session" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		query := r.URL.Query()
		if query.Get("appid") != "app id/+" || query.Get("secret") != "app secret/+" || query.Get("js_code") != "login code/+" || query.Get("grant_type") != "authorization_code" {
			t.Fatalf("query = %q", r.URL.RawQuery)
		}
		writeJSON(t, w, map[string]any{"openid": "openid-1", "session_key": "session-key-1", "unionid": "union-1"})
	}))
	defer server.Close()

	identity, err := NewClient("app id/+", "app secret/+", server.URL, time.Second).ExchangeLoginCode(context.Background(), "login code/+")
	if err != nil {
		t.Fatal(err)
	}
	if identity != (LoginIdentity{OpenID: "openid-1", UnionID: "union-1", SessionKey: "session-key-1"}) {
		t.Fatalf("identity = %+v", identity)
	}
}

func TestClientExchangeLoginCodeAllowsMissingUnionID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(t, w, map[string]any{"openid": "openid-1", "session_key": "session-key-1"})
	}))
	defer server.Close()

	identity, err := NewClient("app", "secret", server.URL, time.Second).ExchangeLoginCode(context.Background(), "code")
	if err != nil || identity.OpenID != "openid-1" || identity.UnionID != "" {
		t.Fatalf("identity = %+v, err = %v", identity, err)
	}
}

func TestClientExchangePhoneCodeCachesAccessTokenUntilExpiry(t *testing.T) {
	var tokenCalls atomic.Int32
	var phoneCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/cgi-bin/token":
			tokenCalls.Add(1)
			if query := r.URL.Query(); query.Get("grant_type") != "client_credential" || query.Get("appid") != "app" || query.Get("secret") != "secret" {
				t.Fatalf("token query = %q", r.URL.RawQuery)
			}
			writeJSON(t, w, map[string]any{"access_token": "access-token", "expires_in": 7200})
		case "/wxa/business/getuserphonenumber":
			phoneCalls.Add(1)
			if r.Method != http.MethodPost || r.URL.Query().Get("access_token") != "access-token" {
				t.Fatalf("phone request = %s %s", r.Method, r.URL.String())
			}
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body["code"] != "phone-code" {
				t.Fatalf("phone body = %#v, err = %v", body, err)
			}
			writeJSON(t, w, map[string]any{
				"errcode": 0,
				"errmsg":  "ok",
				"phone_info": map[string]any{
					"phoneNumber":     "13812345678",
					"purePhoneNumber": "13812345678",
					"countryCode":     "86",
					"watermark":       map[string]any{"timestamp": 1, "appid": "app"},
				},
			})
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient("app", "secret", server.URL, time.Second)
	for range 2 {
		phone, err := client.ExchangePhoneCode(context.Background(), "phone-code")
		if err != nil || phone.Number != "+8613812345678" {
			t.Fatalf("phone = %+v, err = %v", phone, err)
		}
	}
	if tokenCalls.Load() != 1 || phoneCalls.Load() != 2 {
		t.Fatalf("token calls = %d, phone calls = %d", tokenCalls.Load(), phoneCalls.Load())
	}
}

func TestClientExchangePhoneCodeRefreshesExpiredAccessToken(t *testing.T) {
	var tokenCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/cgi-bin/token":
			call := tokenCalls.Add(1)
			writeJSON(t, w, map[string]any{"access_token": "access-token-" + string(rune('0'+call)), "expires_in": 1})
		case "/wxa/business/getuserphonenumber":
			writeJSON(t, w, map[string]any{"errcode": 0, "errmsg": "ok", "phone_info": map[string]any{"phoneNumber": "13812345678", "purePhoneNumber": "13812345678", "countryCode": "86", "watermark": map[string]any{"timestamp": 1, "appid": "app"}}})
		}
	}))
	defer server.Close()

	now := time.Date(2026, 9, 14, 8, 0, 0, 0, time.UTC)
	client := NewClient("app", "secret", server.URL, time.Second).(*apiClient)
	client.now = func() time.Time { return now }
	if _, err := client.ExchangePhoneCode(context.Background(), "phone-code"); err != nil {
		t.Fatal(err)
	}
	now = now.Add(2 * time.Second)
	if _, err := client.ExchangePhoneCode(context.Background(), "phone-code"); err != nil {
		t.Fatal(err)
	}
	if tokenCalls.Load() != 2 {
		t.Fatalf("token calls = %d", tokenCalls.Load())
	}
}

func TestClientExchangePhoneCodeRejectsMissingPhone(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/cgi-bin/token":
			writeJSON(t, w, map[string]any{"access_token": "access-token", "expires_in": 7200})
		case "/wxa/business/getuserphonenumber":
			writeJSON(t, w, map[string]any{"errcode": 0, "errmsg": "ok"})
		}
	}))
	defer server.Close()

	_, err := NewClient("app", "secret", server.URL, time.Second).ExchangePhoneCode(context.Background(), "phone-code")
	if KindOf(err) != ErrorInvalidResponse {
		t.Fatalf("error = %v, kind = %q", err, KindOf(err))
	}
}

func TestClientExchangePhoneCodeRefreshesInvalidAccessTokenOnce(t *testing.T) {
	var tokenCalls atomic.Int32
	var phoneCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/cgi-bin/token":
			call := tokenCalls.Add(1)
			writeJSON(t, w, map[string]any{"access_token": "access-token-" + string(rune('0'+call)), "expires_in": 7200})
		case "/wxa/business/getuserphonenumber":
			phoneCalls.Add(1)
			if r.URL.Query().Get("access_token") == "access-token-1" {
				writeJSON(t, w, map[string]any{"errcode": 40001, "errmsg": "invalid credential"})
				return
			}
			writeJSON(t, w, map[string]any{"errcode": 0, "errmsg": "ok", "phone_info": map[string]any{"phoneNumber": "+8613812345678", "purePhoneNumber": "13812345678", "countryCode": "86", "watermark": map[string]any{"timestamp": 1, "appid": "app"}}})
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	phone, err := NewClient("app", "secret", server.URL, time.Second).ExchangePhoneCode(context.Background(), "phone-code")
	if err != nil || phone.Number != "+8613812345678" {
		t.Fatalf("phone = %+v, err = %v", phone, err)
	}
	if tokenCalls.Load() != 2 || phoneCalls.Load() != 2 {
		t.Fatalf("token calls = %d, phone calls = %d", tokenCalls.Load(), phoneCalls.Load())
	}
}

func TestClientMapsLoginErrors(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		want   ErrorKind
	}{
		{name: "invalid code", status: http.StatusOK, body: `{"errcode":40029,"errmsg":"invalid code"}`, want: ErrorInvalidCredential},
		{name: "rate limited", status: http.StatusOK, body: `{"errcode":45011,"errmsg":"rate limit"}`, want: ErrorRateLimited},
		{name: "business error", status: http.StatusOK, body: `{"errcode":-1,"errmsg":"busy"}`, want: ErrorUnavailable},
		{name: "http rate limited", status: http.StatusTooManyRequests, body: `busy`, want: ErrorRateLimited},
		{name: "non 2xx", status: http.StatusBadGateway, body: `bad gateway`, want: ErrorUnavailable},
		{name: "non json", status: http.StatusOK, body: `not-json`, want: ErrorInvalidResponse},
		{name: "missing field", status: http.StatusOK, body: `{"openid":"openid-only"}`, want: ErrorInvalidResponse},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(test.status)
				_, _ = w.Write([]byte(test.body))
			}))
			defer server.Close()

			_, err := NewClient("app", "secret", server.URL, time.Second).ExchangeLoginCode(context.Background(), "login-code")
			if KindOf(err) != test.want {
				t.Fatalf("error = %v, kind = %q", err, KindOf(err))
			}
		})
	}
}

func TestClientMapsTimeoutToUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
		writeJSON(t, w, map[string]any{"openid": "openid", "session_key": "session-key"})
	}))
	defer server.Close()

	_, err := NewClient("app", "secret", server.URL, 10*time.Millisecond).ExchangeLoginCode(context.Background(), "login-code")
	if KindOf(err) != ErrorUnavailable {
		t.Fatalf("error = %v, kind = %q", err, KindOf(err))
	}
}

func TestClientErrorsAndLogsDoNotExposeSecrets(t *testing.T) {
	const (
		appSecret   = "app-secret-sensitive"
		loginCode   = "login-code-sensitive"
		phoneCode   = "phone-code-sensitive"
		sessionKey  = "session-key-sensitive"
		accessToken = "access-token-sensitive"
		phoneNumber = "+8613912345678"
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sns/jscode2session":
			writeJSON(t, w, map[string]any{"errcode": -1, "errmsg": loginCode + sessionKey + appSecret})
		case "/cgi-bin/token":
			writeJSON(t, w, map[string]any{"access_token": accessToken, "expires_in": 7200})
		case "/wxa/business/getuserphonenumber":
			writeJSON(t, w, map[string]any{"errcode": -1, "errmsg": phoneCode + phoneNumber + accessToken})
		}
	}))
	defer server.Close()

	var logs bytes.Buffer
	originalWriter := log.Writer()
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(originalWriter) })
	client := NewClient("app", appSecret, server.URL, time.Second)
	_, loginErr := client.ExchangeLoginCode(context.Background(), loginCode)
	_, phoneErr := client.ExchangePhoneCode(context.Background(), phoneCode)
	combined := errors.Join(loginErr, phoneErr).Error() + logs.String()
	for _, secret := range []string{appSecret, loginCode, phoneCode, sessionKey, accessToken, phoneNumber} {
		if strings.Contains(combined, secret) {
			t.Fatalf("sensitive value %q leaked in %q", secret, combined)
		}
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatal(err)
	}
}
