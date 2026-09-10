package sms

import "context"

type Sender interface {
	Send(ctx context.Context, phone, purpose, code string) error
}
