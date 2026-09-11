package analysis

import (
	"context"
	"time"
)

type Metadata struct {
	Model, PromptVersion, ResponseStatus string
	Duration                             time.Duration
}

type Analyzer interface {
	Analyze(context.Context, []byte) (Result, Metadata, error)
}

type Failure struct {
	Code      string
	Retryable bool
	Err       error
}

func (e *Failure) Error() string { return e.Code + ": " + e.Err.Error() }
func (e *Failure) Unwrap() error { return e.Err }
