package analysis

import (
	"context"
	"crypto/rand"
	"errors"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/GnEveLynn/FormTally/server/internal/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresDraftRoundTripIsolationAndDiscard(t *testing.T) {
	pool := analysisTestPool(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `insert into users(id,phone) values('user_analysis','+8613800000077'),('user_other','+8613800000066')`); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 11, 8, 0, 0, 0, time.UTC)
	store := NewPostgresStore(pool)
	draft := Draft{ID: "analysis_1", UserID: "user_analysis", ImageKey: strings.Repeat("a", 48), ImageWidth: 100, ImageHeight: 80, OccurredAt: now, LocalDate: "2026-09-11", MealType: "lunch", ProcessingMode: "ai", Status: "processing", Result: Result{Items: []Item{}}, ExpiresAt: now.Add(24 * time.Hour), Revision: 1, CreatedAt: now, UpdatedAt: now}
	if err := store.Insert(ctx, draft); err != nil {
		t.Fatal(err)
	}
	warning := "请核对份量"
	draft.Status = "review_required"
	draft.Result = Result{Items: []Item{{Name: "米饭", Grams: 100, EnergyKcal: 116, ProteinGrams: 2.6, CarbGrams: 25.9, FatGrams: .3, Confidence: "high"}}, Warning: &warning}
	draft.Metadata = Metadata{Model: "test-model", PromptVersion: PromptVersion, ResponseStatus: "completed", Duration: 125 * time.Millisecond}
	if err := store.Update(ctx, draft); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(ctx, "user_analysis", draft.ID)
	if err != nil || got == nil || got.Status != "review_required" || len(got.Result.Items) != 1 || got.Metadata.Duration != 125*time.Millisecond {
		t.Fatalf("got=%+v err=%v", got, err)
	}
	if hidden, err := store.Get(ctx, "user_other", draft.ID); err != nil || hidden != nil {
		t.Fatalf("cross-user got=%+v err=%v", hidden, err)
	}
	if _, err := store.Discard(ctx, "user_analysis", draft.ID, 2); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("revision error=%v", err)
	}
	key, err := store.Discard(ctx, "user_analysis", draft.ID, 1)
	if err != nil || key != draft.ImageKey {
		t.Fatalf("key=%q err=%v", key, err)
	}
}

func analysisTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	base := os.Getenv("TEST_DATABASE_URL")
	if base == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	admin, err := postgres.Open(context.Background(), base)
	if err != nil {
		t.Fatal(err)
	}
	database := "test_analysis_" + strings.ToLower(rand.Text())
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
