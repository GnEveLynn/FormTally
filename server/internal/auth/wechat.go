package auth

import (
	"context"
	"crypto/rand"
	"net/http"
	"time"

	"github.com/GnEveLynn/FormTally/server/internal/wechat"
)

func (s *Service) CreateWeChatSession(ctx context.Context, input WeChatSessionInput) (WeChatSessionResult, error) {
	if input.LoginCode == "" {
		return WeChatSessionResult{}, &Error{Code: "VALIDATION_FAILED", Message: "微信登录凭证不能为空", Status: http.StatusUnprocessableEntity}
	}
	if s.weChat == nil {
		return WeChatSessionResult{}, weChatUnavailable()
	}
	identity, err := s.weChat.ExchangeLoginCode(ctx, input.LoginCode)
	if err != nil {
		return WeChatSessionResult{}, mapWeChatError(err, "WECHAT_CODE_INVALID")
	}
	identity.SessionKey = ""

	token, hash, now, expires := s.newSession()
	session, found, err := s.store.CreateSessionForWeChatIdentity(ctx, identity.OpenID, hash, now, expires)
	if err != nil {
		return WeChatSessionResult{}, err
	}
	if found {
		return WeChatSessionResult{Session: session, Token: token}, nil
	}

	bindingTicket := "wxbind_" + rand.Text() + rand.Text()
	if err := s.store.CreateWeChatBindingTicket(ctx, tokenHash(bindingTicket), identity.OpenID, identity.UnionID, now, now.Add(5*time.Minute)); err != nil {
		return WeChatSessionResult{}, err
	}
	return WeChatSessionResult{BindingRequired: true, BindingTicket: bindingTicket, ExpiresInSeconds: 300}, nil
}

func (s *Service) BindWeChatPhone(ctx context.Context, input WeChatPhoneBindingInput) (SessionResult, string, error) {
	if input.TermsVersion != CurrentTermsVersion || input.PrivacyVersion != CurrentPrivacyVersion {
		return SessionResult{}, "", &Error{Code: "AGREEMENT_VERSION_OUTDATED", Message: "协议版本已更新，请重新确认", Status: http.StatusConflict}
	}
	if input.BindingTicket == "" {
		return SessionResult{}, "", &Error{Code: "WECHAT_BINDING_TICKET_INVALID", Message: "微信绑定凭证无效", Status: http.StatusUnprocessableEntity}
	}
	if input.PhoneCode == "" {
		return SessionResult{}, "", &Error{Code: "WECHAT_PHONE_UNAVAILABLE", Message: "无法获取微信手机号", Status: http.StatusUnprocessableEntity}
	}
	if s.weChat == nil {
		return SessionResult{}, "", weChatUnavailable()
	}
	phone, err := s.weChat.ExchangePhoneCode(ctx, input.PhoneCode)
	if err != nil {
		return SessionResult{}, "", mapWeChatError(err, "WECHAT_PHONE_UNAVAILABLE")
	}
	if !validMainlandPhone(phone.Number) {
		return SessionResult{}, "", &Error{Code: "WECHAT_PHONE_UNAVAILABLE", Message: "无法获取支持的微信手机号", Status: http.StatusUnprocessableEntity}
	}
	token, hash, now, expires := s.newSession()
	result, err := s.store.BindWeChatPhoneAndCreateSession(ctx, tokenHash(input.BindingTicket), phone.Number, input.TermsVersion, input.PrivacyVersion, hash, now, expires)
	return result, token, err
}

func mapWeChatError(err error, invalidCode string) error {
	switch wechat.KindOf(err) {
	case wechat.ErrorInvalidCredential:
		return &Error{Code: invalidCode, Message: "微信凭证无效", Status: http.StatusUnprocessableEntity}
	case wechat.ErrorRateLimited:
		return &Error{Code: "WECHAT_RATE_LIMITED", Message: "微信服务请求过于频繁", Status: http.StatusTooManyRequests}
	default:
		return weChatUnavailable()
	}
}

func weChatUnavailable() error {
	return &Error{Code: "WECHAT_UNAVAILABLE", Message: "微信服务暂时不可用", Status: http.StatusServiceUnavailable}
}
