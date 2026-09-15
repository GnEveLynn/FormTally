package account

import (
	"context"
	"crypto/rand"
	"errors"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/GnEveLynn/FormTally/server/internal/auth"
	"github.com/GnEveLynn/FormTally/server/internal/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestDeleteRejectsWrongPurposeExpiredAndConsumedCodes(t *testing.T) {
	pool := accountTestPool(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 11, 7, 0, 0, 0, time.UTC)
	cases := []struct {
		name, userID, phone, purpose string
		expires                      time.Time
		consumed                     *time.Time
		want                         error
	}{
		{"wrong user", "other_user", "+8613812345601", "delete_account", now.Add(time.Minute), nil, ErrInvalidCode},
		{"wrong purpose", "user_purpose", "+8613812345602", "login", now.Add(time.Minute), nil, ErrInvalidCode},
		{"expired", "user_expired", "+8613812345603", "delete_account", now.Add(-time.Minute), nil, ErrExpiredCode},
		{"consumed", "user_consumed", "+8613812345604", "delete_account", now.Add(time.Minute), &now, ErrInvalidCode},
	}
	for index, test := range cases {
		owner := test.userID
		if test.name == "wrong user" {
			owner = "owner_user"
		}
		if _, err := pool.Exec(ctx, `insert into users(id,phone) values($1,$2)`, owner, test.phone); err != nil {
			t.Fatal(err)
		}
		requestID := "verify_case_" + string(rune('a'+index))
		if _, err := pool.Exec(ctx, `insert into login_codes(id,phone,purpose,code_hash,expires_at,retry_after,consumed_at) values($1,$2,$3,$4,$5,$6,$7)`, requestID, test.phone, test.purpose, auth.VerificationCodeHash(requestID, "123456"), test.expires, now, test.consumed); err != nil {
			t.Fatal(err)
		}
		service := NewService(NewPostgresStore(pool))
		service.now = func() time.Time { return now }
		_, err := service.Delete(ctx, test.userID, DeleteInput{Code: "123456", VerificationRequestID: requestID, Confirmation: "DELETE"})
		if !errors.Is(err, test.want) {
			t.Errorf("%s: err=%v want=%v", test.name, err, test.want)
		}
	}
}

func TestDeleteConsumesOwnedCodeRevokesDataAndQueuesImages(t *testing.T) {
	pool := accountTestPool(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 11, 7, 0, 0, 0, time.UTC)
	statements := []struct {
		query string
		args  []any
	}{
		{`insert into users(id,phone) values('user_delete','+8613812345678')`, nil},
		{`insert into sessions(id,user_id,token_hash,expires_at) values('session_delete','user_delete',decode('00','hex'),$1)`, []any{now.Add(time.Hour)}},
		{`insert into profiles(user_id,biological_sex,birth_date,height_cm,weight_kg,activity_level,timezone,revision,updated_at) values('user_delete','male','1995-06-18',178,72.5,'moderate','Asia/Shanghai',1,$1)`, []any{now}},
		{`insert into meal_analyses(id,user_id,image_key,image_width,image_height,occurred_at,local_date,meal_type,processing_mode,status,expires_at,created_at,updated_at) values('analysis_delete','user_delete','image_delete',1,1,$1,'2026-09-11','lunch','manual','failed',$2,$1,$1)`, []any{now, now.Add(time.Hour)}},
	}
	for _, statement := range statements {
		if _, err := pool.Exec(ctx, statement.query, statement.args...); err != nil {
			t.Fatal(err)
		}
	}
	var err error
	code := "123456"
	_, err = pool.Exec(ctx, `insert into login_codes(id,phone,purpose,code_hash,expires_at,retry_after) values('verify_delete','+8613812345678','delete_account',$1,$2,$3)`, auth.VerificationCodeHash("verify_delete", code), now.Add(time.Minute), now)
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(NewPostgresStore(pool))
	service.now = func() time.Time { return now }
	if _, err = service.Delete(ctx, "user_delete", DeleteInput{Code: code, VerificationRequestID: "verify_delete", Confirmation: "DELETE"}); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"users", "sessions", "profiles", "meal_analyses"} {
		var count int
		if err = pool.QueryRow(ctx, `select count(*) from `+table+` where `+map[string]string{"users": "id", "sessions": "user_id", "profiles": "user_id", "meal_analyses": "user_id"}[table]+`='user_delete'`).Scan(&count); err != nil || count != 0 {
			t.Fatalf("%s count=%d err=%v", table, count, err)
		}
	}
	var queued int
	if err = pool.QueryRow(ctx, `select count(*) from object_deletions where object_key='image_delete'`).Scan(&queued); err != nil || queued != 1 {
		t.Fatalf("queued=%d err=%v", queued, err)
	}
}

func TestDeleteWithWeChatIdentityRejectsAnotherUserAndDeletesOwner(t *testing.T) {
	pool := accountTestPool(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 15, 1, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `insert into users(id,phone) values('wx-owner',null),('wx-other',null); insert into user_identities(id,user_id,provider,provider_subject) values('wx-id','wx-owner','wechat_miniprogram','openid-owner')`); err != nil {
		t.Fatal(err)
	}
	store := NewPostgresStore(pool)
	if _, err := store.DeleteWithWeChatIdentity(ctx, "wx-other", "openid-owner", now); !errors.Is(err, ErrWeChatIdentityMismatch) {
		t.Fatalf("cross-user err=%v", err)
	}
	if _, err := store.DeleteWithWeChatIdentity(ctx, "wx-owner", "openid-owner", now); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := pool.QueryRow(ctx, `select count(*) from users where id='wx-owner'`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("count=%d err=%v", count, err)
	}
}

func accountTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	base := os.Getenv("TEST_DATABASE_URL")
	if base == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	admin, err := postgres.Open(context.Background(), base)
	if err != nil {
		t.Fatal(err)
	}
	database := "test_account_" + strings.ToLower(rand.Text())
	if _, err = admin.Exec(context.Background(), "create database "+pgx.Identifier{database}.Sanitize()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = admin.Exec(context.Background(), "drop database "+pgx.Identifier{database}.Sanitize()+" with (force)")
		admin.Close()
	})
	parsed, _ := url.Parse(base)
	parsed.Path = "/" + database
	if err = postgres.Migrate(context.Background(), parsed.String(), "../../migrations", "up"); err != nil {
		t.Fatal(err)
	}
	pool, err := postgres.Open(context.Background(), parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}
