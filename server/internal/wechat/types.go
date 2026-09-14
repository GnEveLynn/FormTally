package wechat

import (
	"context"
	"errors"
)

type LoginIdentity struct {
	OpenID     string
	UnionID    string
	SessionKey string
}

type Phone struct{ Number string }

type Client interface {
	ExchangeLoginCode(context.Context, string) (LoginIdentity, error)
	ExchangePhoneCode(context.Context, string) (Phone, error)
}

type ErrorKind string

const (
	ErrorInvalidCredential ErrorKind = "invalid_credential"
	ErrorRateLimited       ErrorKind = "rate_limited"
	ErrorUnavailable       ErrorKind = "unavailable"
	ErrorInvalidResponse   ErrorKind = "invalid_response"
)

type Error struct{ Kind ErrorKind }

func (e *Error) Error() string { return string(e.Kind) }

func KindOf(err error) ErrorKind {
	var providerError *Error
	if errors.As(err, &providerError) {
		return providerError.Kind
	}
	return ""
}
