package wechat

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const maxResponseBytes = 1 << 20

type apiClient struct {
	appID      string
	appSecret  string
	baseURL    string
	httpClient *http.Client
	now        func() time.Time

	tokenMu      sync.Mutex
	accessToken  string
	tokenExpires time.Time
}

func NewClient(appID, appSecret, baseURL string, timeout time.Duration) Client {
	return &apiClient{
		appID: appID, appSecret: appSecret, baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: timeout}, now: time.Now,
	}
}

func (c *apiClient) ExchangeLoginCode(ctx context.Context, code string) (LoginIdentity, error) {
	query := url.Values{
		"appid":      {c.appID},
		"secret":     {c.appSecret},
		"js_code":    {code},
		"grant_type": {"authorization_code"},
	}
	var response struct {
		providerResponse
		OpenID     string `json:"openid"`
		UnionID    string `json:"unionid"`
		SessionKey string `json:"session_key"`
	}
	if err := c.getJSON(ctx, "/sns/jscode2session?"+query.Encode(), &response); err != nil {
		return LoginIdentity{}, err
	}
	if response.ErrCode != 0 {
		return LoginIdentity{}, classifyProviderError(response.ErrCode)
	}
	if response.OpenID == "" || response.SessionKey == "" {
		return LoginIdentity{}, providerError(ErrorInvalidResponse)
	}
	return LoginIdentity{OpenID: response.OpenID, UnionID: response.UnionID, SessionKey: response.SessionKey}, nil
}

func (c *apiClient) ExchangePhoneCode(ctx context.Context, code string) (Phone, error) {
	token, err := c.getAccessToken(ctx, false)
	if err != nil {
		return Phone{}, err
	}
	phone, invalidToken, err := c.exchangePhoneCode(ctx, token, code)
	if !invalidToken {
		return phone, err
	}
	token, err = c.getAccessToken(ctx, true)
	if err != nil {
		return Phone{}, err
	}
	phone, invalidToken, err = c.exchangePhoneCode(ctx, token, code)
	if invalidToken {
		return Phone{}, providerError(ErrorUnavailable)
	}
	return phone, err
}

func (c *apiClient) getAccessToken(ctx context.Context, force bool) (string, error) {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()
	if !force && c.accessToken != "" && c.now().Before(c.tokenExpires) {
		return c.accessToken, nil
	}

	query := url.Values{
		"grant_type": {"client_credential"},
		"appid":      {c.appID},
		"secret":     {c.appSecret},
	}
	var response struct {
		providerResponse
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err := c.getJSON(ctx, "/cgi-bin/token?"+query.Encode(), &response); err != nil {
		return "", err
	}
	if response.ErrCode != 0 {
		return "", classifyProviderError(response.ErrCode)
	}
	if response.AccessToken == "" || response.ExpiresIn <= 0 {
		return "", providerError(ErrorInvalidResponse)
	}
	c.accessToken = response.AccessToken
	c.tokenExpires = c.now().Add(time.Duration(response.ExpiresIn) * time.Second)
	return c.accessToken, nil
}

func (c *apiClient) exchangePhoneCode(ctx context.Context, token, code string) (Phone, bool, error) {
	body, err := json.Marshal(struct {
		Code string `json:"code"`
	}{Code: code})
	if err != nil {
		return Phone{}, false, providerError(ErrorInvalidResponse)
	}
	query := url.Values{"access_token": {token}}
	var response struct {
		providerResponse
		PhoneInfo struct {
			PhoneNumber     string `json:"phoneNumber"`
			PurePhoneNumber string `json:"purePhoneNumber"`
			CountryCode     string `json:"countryCode"`
		} `json:"phone_info"`
	}
	if err := c.postJSON(ctx, "/wxa/business/getuserphonenumber?"+query.Encode(), body, &response); err != nil {
		return Phone{}, false, err
	}
	if isInvalidAccessToken(response.ErrCode) {
		return Phone{}, true, nil
	}
	if response.ErrCode != 0 {
		return Phone{}, false, classifyProviderError(response.ErrCode)
	}
	number := response.PhoneInfo.PhoneNumber
	if response.PhoneInfo.CountryCode == "86" && response.PhoneInfo.PurePhoneNumber != "" {
		number = "+86" + response.PhoneInfo.PurePhoneNumber
	}
	if number == "" {
		return Phone{}, false, providerError(ErrorInvalidResponse)
	}
	return Phone{Number: number}, false, nil
}

type providerResponse struct {
	ErrCode int `json:"errcode"`
}

func (c *apiClient) getJSON(ctx context.Context, path string, destination any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return providerError(ErrorUnavailable)
	}
	return c.doJSON(request, destination)
}

func (c *apiClient) postJSON(ctx context.Context, path string, body []byte, destination any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, strings.NewReader(string(body)))
	if err != nil {
		return providerError(ErrorUnavailable)
	}
	request.Header.Set("Content-Type", "application/json")
	return c.doJSON(request, destination)
}

func (c *apiClient) doJSON(request *http.Request, destination any) error {
	response, err := c.httpClient.Do(request)
	if err != nil {
		return providerError(ErrorUnavailable)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		if response.StatusCode == http.StatusTooManyRequests {
			return providerError(ErrorRateLimited)
		}
		return providerError(ErrorUnavailable)
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, maxResponseBytes))
	if err := decoder.Decode(destination); err != nil {
		return providerError(ErrorInvalidResponse)
	}
	return nil
}

func classifyProviderError(code int) error {
	switch code {
	case 40029, 40163:
		return providerError(ErrorInvalidCredential)
	case 45009, 45011:
		return providerError(ErrorRateLimited)
	default:
		return providerError(ErrorUnavailable)
	}
}

func isInvalidAccessToken(code int) bool {
	return code == 40001 || code == 40014 || code == 42001
}

func providerError(kind ErrorKind) error { return &Error{Kind: kind} }
