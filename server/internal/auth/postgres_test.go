package auth

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

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
	if _, err := service.store.pool.Exec(ctx, `insert into profiles(user_id,biological_sex,birth_date,height_cm,weight_kg,activity_level,timezone,revision,updated_at) values($1,'male','1995-06-18',178,72.5,'moderate','Asia/Shanghai',1,now())`, created.User.ID); err != nil {
		t.Fatal(err)
	}
	current, err = service.GetSession(ctx, token)
	if err != nil || current.User.OnboardingStatus != "goal_required" {
		t.Fatalf("profile session = %+v, %v", current, err)
	}
	if _, err := service.store.pool.Exec(ctx, `insert into goal_settings(user_id,mode,objective,pace,revision,updated_at) values($1,'automatic','maintain',null,1,now())`, created.User.ID); err != nil {
		t.Fatal(err)
	}
	current, err = service.GetSession(ctx, token)
	if err != nil || current.User.OnboardingStatus != "completed" {
		t.Fatalf("completed session = %+v, %v", current, err)
	}
	if err := service.RevokeSession(ctx, token); err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetSession(ctx, token); ErrorCode(err) != "UNAUTHENTICATED" {
		t.Fatalf("revoked session error = %v", err)
	}
}

func TestWeChatMigrationConstraintsAndCascades(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()
	pool := service.store.pool
	if _, err := pool.Exec(ctx, `insert into users(id,phone) values ('user-wechat-1','+8613812345678'),('user-wechat-2','+8613912345678')`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `insert into user_identities(id,user_id,provider,provider_subject) values ('identity-1','user-wechat-1','wechat_miniprogram','openid-1')`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `insert into user_identities(id,user_id,provider,provider_subject) values ('identity-2','user-wechat-2','wechat_miniprogram','openid-1')`); err == nil {
		t.Fatal("duplicate WeChat identity accepted")
	}
	if _, err := pool.Exec(ctx, `insert into sessions(id,user_id,token_hash,expires_at) values ('session-web','user-wechat-1',decode('01','hex'),now()+interval '1 hour')`); err != nil {
		t.Fatal(err)
	}
	var clientType string
	if err := pool.QueryRow(ctx, `select client_type from sessions where id='session-web'`).Scan(&clientType); err != nil || clientType != "web" {
		t.Fatalf("client_type = %q, err = %v", clientType, err)
	}
	if _, err := pool.Exec(ctx, `insert into sessions(id,user_id,token_hash,expires_at,client_type) values ('session-invalid','user-wechat-1',decode('02','hex'),now()+interval '1 hour','desktop')`); err == nil {
		t.Fatal("invalid session client_type accepted")
	}
	if _, err := pool.Exec(ctx, `insert into wechat_binding_tickets(token_hash,openid,expires_at,user_id) values (decode('03','hex'),'openid-1',now()+interval '5 minutes','user-wechat-1')`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `delete from users where id='user-wechat-1'`); err != nil {
		t.Fatal(err)
	}
	var identities, tickets int
	if err := pool.QueryRow(ctx, `select (select count(*) from user_identities where user_id='user-wechat-1'),(select count(*) from wechat_binding_tickets where user_id='user-wechat-1')`).Scan(&identities, &tickets); err != nil {
		t.Fatal(err)
	}
	if identities != 0 || tickets != 0 {
		t.Fatalf("cascade counts = identities %d, tickets %d", identities, tickets)
	}
}

func TestWeChatBindingTicketStoresOnlyHashAndExpires(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 14, 8, 0, 0, 0, time.UTC)
	raw := "bind-sensitive-token"
	hash := sha256.Sum256([]byte(raw))
	if err := service.store.CreateWeChatBindingTicket(ctx, hash[:], "openid-1", "union-1", now, now.Add(5*time.Minute)); err != nil {
		t.Fatal(err)
	}
	var stored []byte
	var expires time.Time
	if err := service.store.pool.QueryRow(ctx, `select token_hash,expires_at from wechat_binding_tickets where openid='openid-1'`).Scan(&stored, &expires); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(stored, hash[:]) || bytes.Contains(stored, []byte(raw)) || !expires.Equal(now.Add(5*time.Minute)) {
		t.Fatalf("stored hash = %x, expires = %s", stored, expires)
	}
	_, err := service.store.BindWeChatPhoneAndCreateSession(ctx, hash[:], "+8613812345678", CurrentTermsVersion, CurrentPrivacyVersion, []byte("session-hash"), now.Add(6*time.Minute), now.Add(30*24*time.Hour))
	if ErrorCode(err) != "WECHAT_BINDING_TICKET_EXPIRED" {
		t.Fatalf("expired ticket error = %v", err)
	}
}

func TestWeChatBindingTicketCanOnlyBeConsumedOnce(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 14, 8, 0, 0, 0, time.UTC)
	hash := sha256.Sum256([]byte("binding-ticket"))
	if err := service.store.CreateWeChatBindingTicket(ctx, hash[:], "openid-1", "", now, now.Add(5*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := service.store.BindWeChatPhoneAndCreateSession(ctx, hash[:], "+8613812345678", CurrentTermsVersion, CurrentPrivacyVersion, []byte("session-hash-1"), now, now.Add(30*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := service.store.BindWeChatPhoneAndCreateSession(ctx, hash[:], "+8613812345678", CurrentTermsVersion, CurrentPrivacyVersion, []byte("session-hash-2"), now, now.Add(30*24*time.Hour)); ErrorCode(err) != "WECHAT_BINDING_TICKET_CONSUMED" {
		t.Fatalf("second consume error = %v", err)
	}
}

func TestWeChatBindingTicketConcurrentConsumeHasOneWinner(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 14, 8, 0, 0, 0, time.UTC)
	hash := sha256.Sum256([]byte("concurrent-binding-ticket"))
	if err := service.store.CreateWeChatBindingTicket(ctx, hash[:], "openid-1", "", now, now.Add(5*time.Minute)); err != nil {
		t.Fatal(err)
	}

	start := make(chan struct{})
	errorsByCall := make([]error, 2)
	var wait sync.WaitGroup
	for index := range errorsByCall {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			_, errorsByCall[index] = service.store.BindWeChatPhoneAndCreateSession(ctx, hash[:], "+8613812345678", CurrentTermsVersion, CurrentPrivacyVersion, []byte{byte(index + 1)}, now, now.Add(30*24*time.Hour))
		}()
	}
	close(start)
	wait.Wait()

	successes, consumed := 0, 0
	for _, err := range errorsByCall {
		if err == nil {
			successes++
		} else if ErrorCode(err) == "WECHAT_BINDING_TICKET_CONSUMED" {
			consumed++
		} else {
			t.Fatalf("unexpected consume error = %v", err)
		}
	}
	if successes != 1 || consumed != 1 {
		t.Fatalf("successes = %d, consumed = %d", successes, consumed)
	}
}

func TestWeChatBindingMergesAccountsWithoutSilentConflict(t *testing.T) {
	t.Run("binds new identity to existing phone user", func(t *testing.T) {
		service, _ := testService(t)
		ctx := context.Background()
		now := time.Date(2026, 9, 14, 8, 0, 0, 0, time.UTC)
		if _, err := service.store.pool.Exec(ctx, `insert into users(id,phone) values ('existing-user','+8613812345678')`); err != nil {
			t.Fatal(err)
		}
		result := bindWeChatForTest(t, service.store, "ticket-existing", "openid-existing", "+8613812345678", now)
		if result.User.ID != "existing-user" {
			t.Fatalf("user ID = %q", result.User.ID)
		}
	})

	t.Run("creates user for new phone", func(t *testing.T) {
		service, _ := testService(t)
		now := time.Date(2026, 9, 14, 8, 0, 0, 0, time.UTC)
		result := bindWeChatForTest(t, service.store, "ticket-new", "openid-new", "+8613912345678", now)
		if result.User.ID == "" || result.User.PhoneMasked != "+86 139****5678" {
			t.Fatalf("result = %+v", result)
		}
	})

	t.Run("repeated identity binding is idempotent", func(t *testing.T) {
		service, _ := testService(t)
		now := time.Date(2026, 9, 14, 8, 0, 0, 0, time.UTC)
		first := bindWeChatForTest(t, service.store, "ticket-first", "openid-repeat", "+8613712345678", now)
		second := bindWeChatForTest(t, service.store, "ticket-second", "openid-repeat", "+8613712345678", now.Add(time.Minute))
		if first.User.ID != second.User.ID {
			t.Fatalf("user IDs = %q, %q", first.User.ID, second.User.ID)
		}
	})

	t.Run("rejects identity and phone owned by different users", func(t *testing.T) {
		service, _ := testService(t)
		ctx := context.Background()
		now := time.Date(2026, 9, 14, 8, 0, 0, 0, time.UTC)
		first := bindWeChatForTest(t, service.store, "ticket-owner", "openid-conflict", "+8613612345678", now)
		if _, err := service.store.pool.Exec(ctx, `insert into users(id,phone) values ('other-user','+8613512345678')`); err != nil {
			t.Fatal(err)
		}
		hash := sha256.Sum256([]byte("ticket-conflict"))
		if err := service.store.CreateWeChatBindingTicket(ctx, hash[:], "openid-conflict", "", now, now.Add(5*time.Minute)); err != nil {
			t.Fatal(err)
		}
		if _, err := service.store.BindWeChatPhoneAndCreateSession(ctx, hash[:], "+8613512345678", CurrentTermsVersion, CurrentPrivacyVersion, []byte("conflict-session"), now, now.Add(30*24*time.Hour)); ErrorCode(err) != "IDENTITY_CONFLICT" {
			t.Fatalf("conflict error = %v", err)
		}
		var identityUser string
		if err := service.store.pool.QueryRow(ctx, `select user_id from user_identities where provider='wechat_miniprogram' and provider_subject='openid-conflict'`).Scan(&identityUser); err != nil {
			t.Fatal(err)
		}
		if identityUser != first.User.ID {
			t.Fatalf("identity user changed to %q", identityUser)
		}
		var consumedAt *time.Time
		if err := service.store.pool.QueryRow(ctx, `select consumed_at from wechat_binding_tickets where token_hash=$1`, hash[:]).Scan(&consumedAt); err != nil || consumedAt != nil {
			t.Fatalf("consumed_at = %v, err = %v", consumedAt, err)
		}
	})
}

func bindWeChatForTest(t *testing.T, store *PostgresStore, rawTicket, openID, phone string, now time.Time) SessionResult {
	t.Helper()
	hash := sha256.Sum256([]byte(rawTicket))
	if err := store.CreateWeChatBindingTicket(context.Background(), hash[:], openID, "", now, now.Add(5*time.Minute)); err != nil {
		t.Fatal(err)
	}
	result, err := store.BindWeChatPhoneAndCreateSession(context.Background(), hash[:], phone, CurrentTermsVersion, CurrentPrivacyVersion, []byte("session-"+rawTicket), now, now.Add(30*24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	return result
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
