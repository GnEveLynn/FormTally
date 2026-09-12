package account

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeStore struct{ called bool }

func (f *fakeStore) Delete(context.Context, string, DeleteInput, time.Time) (Deletion, error) {
	f.called = true
	return Deletion{Status: "accepted"}, nil
}

func TestDeleteRequiresExplicitConfirmation(t *testing.T) {
	store := &fakeStore{}
	service := NewService(store)
	_, err := service.Delete(context.Background(), "user_1", DeleteInput{Code: "123456", VerificationRequestID: "verify_1", Confirmation: "delete"})
	if !errors.Is(err, ErrConfirmationRequired) || store.called {
		t.Fatalf("err=%v called=%v", err, store.called)
	}
}
