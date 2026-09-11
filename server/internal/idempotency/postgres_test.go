package idempotency

import (
	"context"
	"crypto/rand"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	appdb "github.com/GnEveLynn/FormTally/server/internal/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresStoreReplaysCompletedRequestAndRejectsChangedPayload(t *testing.T) {
	pool := idempotencyTestPool(t)
	store := NewPostgresStore(pool)
	ctx := context.Background()
	now := time.Date(2026, 9, 11, 8, 0, 0, 0, time.UTC)
	result, err := store.Begin(ctx, "user_idem", "analysis:create", "key-1", "hash-a", now, now.Add(24*time.Hour))
	if err != nil || result.State != Started {
		t.Fatalf("first begin = %+v %v", result, err)
	}
	if _, err := store.Begin(ctx, "user_idem", "analysis:create", "key-1", "hash-a", now, now.Add(24*time.Hour)); err != ErrInProgress {
		t.Fatalf("second begin err = %v", err)
	}
	if err := store.Complete(ctx, "user_idem", "analysis:create", "key-1", 201, []byte(`{"analysis":{"id":"a1"}}`), now); err != nil {
		t.Fatal(err)
	}
	replay, err := store.Begin(ctx, "user_idem", "analysis:create", "key-1", "hash-a", now, now.Add(24*time.Hour))
	if err != nil || replay.State != Replayed || replay.Status != 201 {
		t.Fatalf("replay = %+v %v", replay, err)
	}
	if _, err := store.Begin(ctx, "user_idem", "analysis:create", "key-1", "hash-b", now, now.Add(24*time.Hour)); err != ErrConflict {
		t.Fatalf("changed payload err = %v", err)
	}
}

func idempotencyTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	base := os.Getenv("TEST_DATABASE_URL")
	if base == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	admin, err := appdb.Open(context.Background(), base)
	if err != nil {
		t.Fatal(err)
	}
	database := "test_idem_" + strings.ToLower(rand.Text())
	if _, err := admin.Exec(context.Background(), "create database "+pgx.Identifier{database}.Sanitize()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = admin.Exec(context.Background(), "drop database "+pgx.Identifier{database}.Sanitize()+" with (force)")
		admin.Close()
	})
	parsed, _ := url.Parse(base)
	parsed.Path = "/" + database
	if err := appdb.Migrate(context.Background(), parsed.String(), "../../migrations", "up"); err != nil {
		t.Fatal(err)
	}
	pool, err := appdb.Open(context.Background(), parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.Exec(context.Background(), `insert into users(id,phone) values('user_idem','+8613800000099')`); err != nil {
		t.Fatal(err)
	}
	return pool
}
