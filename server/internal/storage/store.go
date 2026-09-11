package storage

import (
	"context"
	"io"
	"time"
)

type Store interface {
	Put(context.Context, string, io.Reader, string) (string, error)
	Open(context.Context, string) (io.ReadCloser, error)
	PrivateURL(context.Context, string, time.Duration) (string, error)
	Delete(context.Context, string) error
}
