package account

import (
	"context"
	"errors"
	"net/http"
	"time"
)

var (
	ErrConfirmationRequired = errors.New("account deletion confirmation required")
	ErrInvalidCode          = errors.New("invalid account deletion code")
	ErrExpiredCode          = errors.New("expired account deletion code")
)

type DeleteInput struct {
	Code                  string `json:"code"`
	VerificationRequestID string `json:"verificationRequestId"`
	Confirmation          string `json:"confirmation"`
}

type Deletion struct {
	Status          string    `json:"status"`
	AccessRevokedAt time.Time `json:"accessRevokedAt"`
	PurgeBy         time.Time `json:"purgeBy"`
}

type Store interface {
	Delete(context.Context, string, DeleteInput, time.Time) (Deletion, error)
}

type Service struct {
	store Store
	now   func() time.Time
}

func NewService(store Store) *Service { return &Service{store: store, now: time.Now} }

func (s *Service) Delete(ctx context.Context, userID string, input DeleteInput) (Deletion, error) {
	if input.Confirmation != "DELETE" {
		return Deletion{}, ErrConfirmationRequired
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
	default:
		return http.StatusInternalServerError, "INTERNAL_ERROR", "服务暂时不可用"
	}
}
