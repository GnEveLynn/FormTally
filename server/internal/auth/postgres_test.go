package auth

import (
	"context"
	"crypto/rand"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/GnEveLynn/FormTally/server/internal/postgres"
	"github.com/GnEveLynn/FormTally/server/internal/sms"
	"github.com/jackc/pgx/v5"
)

func TestVerificationAndSessionLifecycle(t *testing.T) {
	service, sender := testService(t)
	ctx := context.Background()
	phone := "+8613812345678"

	verification, err := service.RequestCode(ctx, RequestCodeInput{Phone: phone, Purpose: PurposeLogin})
	if err != nil {
		t.Fatal(err)
	}
	if verification.ExpiresInSeconds != 300 || verification.RetryAfterSeconds != 60 {
		t.Fatalf("verification = %+v", verification)
	}
	if _, err := service.RequestCode(ctx, RequestCodeInput{Phone: phone, Purpose: PurposeLogin}); ErrorCode(err) != "RATE_LIMITED" {
		t.Fatalf("second request error = %v", err)
	}
	code, ok := sender.LastCode(phone, string(PurposeLogin))
	if !ok {
		t.Fatal("test sender did not receive code")
	}
	created, token, err := service.CreateSession(ctx, CreateSessionInput{
		Phone: phone, Code: code, VerificationRequestID: verification.RequestID,
		TermsVersion: CurrentTermsVersion, PrivacyVersion: CurrentPrivacyVersion,
	})
	if err != nil {
		t.Fatal(err)
	}
	if token == "" || created.User.OnboardingStatus != "profile_required" || created.User.PhoneMasked != "+86 138****5678" {
		t.Fatalf("created session = %+v, token empty = %v", created, token == "")
	}
	if _, _, err := service.CreateSession(ctx, CreateSessionInput{Phone: phone, Code: code, VerificationRequestID: verification.RequestID, TermsVersion: CurrentTermsVersion, PrivacyVersion: CurrentPrivacyVersion}); ErrorCode(err) != "VERIFICATION_CODE_INVALID" {
		t.Fatalf("reused code error = %v", err)
	}
	current, err := service.GetSession(ctx, token)
	if err != nil || current.User.ID != created.User.ID {
		t.Fatalf("GetSession() = %+v, %v", current, err)
	}
	if err := service.RevokeSession(ctx, token); err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetSession(ctx, token); ErrorCode(err) != "UNAUTHENTICATED" {
		t.Fatalf("revoked session error = %v", err)
	}
}

func testService(t *testing.T) (*Service, *sms.TestSender) {
	t.Helper()
	databaseURL := isolatedAuthDatabase(t)
	if err := postgres.Migrate(context.Background(), databaseURL, "../../migrations", "up"); err != nil {
		t.Fatal(err)
	}
	pool, err := postgres.Open(context.Background(), databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	sender := sms.NewTestSender(nil)
	return NewService(NewPostgresStore(pool), sender), sender
}

func isolatedAuthDatabase(t *testing.T) string {
	t.Helper()
	base := os.Getenv("TEST_DATABASE_URL")
	if base == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	admin, err := postgres.Open(context.Background(), base)
	if err != nil {
		t.Fatal(err)
	}
	database := "test_auth_" + strings.ToLower(rand.Text())
	if _, err := admin.Exec(context.Background(), "create database "+pgx.Identifier{database}.Sanitize()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = admin.Exec(context.Background(), "drop database "+pgx.Identifier{database}.Sanitize()+" with (force)")
		admin.Close()
	})
	parsed, err := url.Parse(base)
	if err != nil {
		t.Fatal(err)
	}
	parsed.Path = "/" + database
	return parsed.String()
}
