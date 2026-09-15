package account

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/GnEveLynn/FormTally/server/internal/wechat"
)

type fakeStore struct{ called bool }

func (f *fakeStore) Delete(context.Context, string, DeleteInput, time.Time) (Deletion, error) {
	f.called = true
	return Deletion{Status: "accepted"}, nil
}
func (f *fakeStore) DeleteWithWeChatIdentity(context.Context, string, string, time.Time) (Deletion, error) {
	f.called = true
	return Deletion{Status: "accepted"}, nil
}

type fakeWeChat struct {
	identity wechat.LoginIdentity
	err      error
}

func (f fakeWeChat) ExchangeLoginCode(context.Context, string) (wechat.LoginIdentity, error) {
	return f.identity, f.err
}
func (fakeWeChat) ExchangePhoneCode(context.Context, string) (wechat.Phone, error) {
	return wechat.Phone{}, nil
}

func TestDeleteRequiresExplicitConfirmation(t *testing.T) {
	store := &fakeStore{}
	service := NewService(store)
	_, err := service.Delete(context.Background(), "user_1", DeleteInput{Code: "123456", VerificationRequestID: "verify_1", Confirmation: "delete"})
	if !errors.Is(err, ErrConfirmationRequired) || store.called {
		t.Fatalf("err=%v called=%v", err, store.called)
	}
}

func TestDeleteWithoutPhoneRequiresFreshMatchingWeChatIdentity(t *testing.T) {
	store := &fakeStore{}
	service := NewService(store, fakeWeChat{identity: wechat.LoginIdentity{OpenID: "openid-user"}})
	_, err := service.Delete(context.Background(), "user_1", DeleteInput{LoginCode: "fresh-code", Confirmation: "DELETE"})
	if err != nil || !store.called {
		t.Fatalf("err=%v called=%v", err, store.called)
	}
}
