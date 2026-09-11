package storage

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"
)

type deletionRepo struct{ completed, retried bool }

func (d *deletionRepo) Next(context.Context, time.Time) (Deletion, bool, error) {
	return Deletion{ID: "delete_1", ObjectKey: "0123456789abcdef0123456789abcdef0123456789abcdef", Attempts: 0}, true, nil
}
func (d *deletionRepo) Complete(context.Context, string, time.Time) error {
	d.completed = true
	return nil
}
func (d *deletionRepo) Retry(context.Context, string, int, time.Time) error {
	d.retried = true
	return nil
}

type failingObjects struct{}

func (failingObjects) Put(context.Context, string, io.Reader, string) (string, error) { return "", nil }
func (failingObjects) Open(context.Context, string) (io.ReadCloser, error)            { return nil, nil }
func (failingObjects) PrivateURL(context.Context, string, time.Duration) (string, error) {
	return "", nil
}
func (failingObjects) Delete(context.Context, string) error { return errors.New("temporary") }
func TestDeletionWorkerPersistsRetryAfterObjectFailure(t *testing.T) {
	repo := &deletionRepo{}
	worker := NewDeletionWorker(repo, failingObjects{})
	worker.now = func() time.Time { return time.Date(2026, 9, 11, 8, 0, 0, 0, time.UTC) }
	worked, err := worker.RunOnce(context.Background())
	if err != nil || !worked || !repo.retried || repo.completed {
		t.Fatalf("worked=%v err=%v repo=%+v", worked, err, repo)
	}
}
