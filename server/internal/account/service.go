package account

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/GnEveLynn/FormTally/server/internal/wechat"
)

var (
	ErrConfirmationRequired   = errors.New("account deletion confirmation required")
	ErrInvalidCode            = errors.New("invalid account deletion code")
	ErrExpiredCode            = errors.New("expired account deletion code")
	ErrWeChatIdentityMismatch = errors.New("wechat identity does not belong to user")
	ErrWeChatUnavailable      = errors.New("wechat verification unavailable")
)

type DeleteInput struct {
	Code                  string `json:"code"`
	VerificationRequestID string `json:"verificationRequestId"`
	LoginCode             string `json:"loginCode"`
	Confirmation          string `json:"confirmation"`
}

type Deletion struct {
	Status          string    `json:"status"`
	AccessRevokedAt time.Time `json:"accessRevokedAt"`
	PurgeBy         time.Time `json:"purgeBy"`
}

type Store interface {
	Delete(context.Context, string, DeleteInput, time.Time) (Deletion, error)
	DeleteWithWeChatIdentity(context.Context, string, string, time.Time) (Deletion, error)
}

type Service struct {
	store  Store
	now    func() time.Time
	weChat wechat.Client
}

func NewService(store Store, clients ...wechat.Client) *Service {
	service := &Service{store: store, now: time.Now}
	if len(clients) > 0 {
		service.weChat = clients[0]
	}
	return service
}

func (s *Service) Delete(ctx context.Context, userID string, input DeleteInput) (Deletion, error) {
	if input.Confirmation != "DELETE" {
		return Deletion{}, ErrConfirmationRequired
	}
	if input.LoginCode != "" {
		if s.weChat == nil {
			return Deletion{}, ErrWeChatUnavailable
		}
		identity, err := s.weChat.ExchangeLoginCode(ctx, input.LoginCode)
		if err != nil || identity.OpenID == "" {
			return Deletion{}, ErrWeChatIdentityMismatch
		}
		return s.store.DeleteWithWeChatIdentity(ctx, userID, identity.OpenID, s.now().UTC())
	}
	if len(input.Code) != 6 || input.VerificationRequestID == "" {
		return Deletion{}, ErrInvalidCode
	}
	return s.store.Delete(ctx, userID, input, s.now().UTC())
}

func status(err error) (int, string, string) {
	switch {
	case errors.Is(err, ErrConfirmationRequired):
		return http.StatusUnprocessableEntity, "CONFIRMATION_REQUIRED", "请输入 DELETE 确认删除账户"
	case errors.Is(err, ErrInvalidCode):
		return http.StatusUnprocessableEntity, "VERIFICATION_CODE_INVALID", "验证码错误"
	case errors.Is(err, ErrExpiredCode):
		return http.StatusGone, "VERIFICATION_CODE_EXPIRED", "验证码已过期"
	case errors.Is(err, ErrWeChatIdentityMismatch):
		return http.StatusUnprocessableEntity, "WECHAT_IDENTITY_MISMATCH", "微信身份验证失败，请重试"
	case errors.Is(err, ErrWeChatUnavailable):
		return http.StatusServiceUnavailable, "WECHAT_UNAVAILABLE", "微信服务暂时不可用"
	default:
		return http.StatusInternalServerError, "INTERNAL_ERROR", "服务暂时不可用"
	}
}
