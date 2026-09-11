package idempotency

import (
	"context"
	"errors"
	"time"
)

var (
	ErrConflict   = errors.New("idempotency key reused with different content")
	ErrInProgress = errors.New("operation in progress")
)

type State string

const (
	Started  State = "started"
	Replayed State = "replayed"
)

type Result struct {
	State  State
	Status int
	Body   []byte
}

type Store interface {
	Begin(context.Context, string, string, string, string, time.Time, time.Time) (Result, error)
	Complete(context.Context, string, string, string, int, []byte, time.Time) error
}
