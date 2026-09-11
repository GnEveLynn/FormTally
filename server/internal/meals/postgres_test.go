package meals

import (
	"context"
	"crypto/rand"
	"errors"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/GnEveLynn/FormTally/server/internal/idempotency"
	"github.com/GnEveLynn/FormTally/server/internal/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestSaveMealAtomicallyConsumesAnalysisAndIsIdempotent(t *testing.T) {
	pool := mealTestPool(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 11, 8, 0, 0, 0, time.UTC)
	_, err := pool.Exec(ctx, `insert into users(id,phone) values('user_meal','+8613800000088')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `insert into meal_analyses(id,user_id,image_key,image_width,image_height,occurred_at,local_date,meal_type,processing_mode,status,expires_at,created_at,updated_at) values('analysis_1','user_meal','0123456789abcdef0123456789abcdef0123456789abcdef',1,1,$1,'2026-09-11','lunch','ai','review_required',$2,$1,$1)`, now, now.Add(24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(NewPostgresStore(pool), idempotency.NewPostgresStore(pool), nil)
	service.now = func() time.Time { return now }
	analysisID := "analysis_1"
	input := CreateInput{AnalysisID: &analysisID, OccurredAt: "2026-09-11T12:00:00+08:00", MealType: "lunch", Items: []ItemInput{{Name: "鸡胸肉", Grams: 120, Nutrition: Nutrition{EnergyKcal: 198, ProteinGrams: 37.2, FatGrams: 4.3}, Origin: "ai_modified"}}}
	first, err := service.Create(ctx, "user_meal", "save-key", input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Create(ctx, "user_meal", "save-key", input)
	if err != nil {
		t.Fatal(err)
	}
	if first.Meal.ID != second.Meal.ID || first.Meal.Totals.EnergyKcal != 198 {
		t.Fatalf("first=%+v second=%+v", first, second)
	}
	var mealsCount int
	if err := pool.QueryRow(ctx, `select count(*) from meals where user_id='user_meal'`).Scan(&mealsCount); err != nil || mealsCount != 1 {
		t.Fatalf("count=%d err=%v", mealsCount, err)
	}
	var status, mealID string
	if err := pool.QueryRow(ctx, `select status,meal_id from meal_analyses where id='analysis_1'`).Scan(&status, &mealID); err != nil || status != "saved" || mealID != first.Meal.ID {
		t.Fatalf("draft status=%s meal=%s err=%v", status, mealID, err)
	}
	if _, err := pool.Exec(ctx, `insert into users(id,phone) values('user_other','+8613800000099')`); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Get(ctx, "user_other", first.Meal.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-user get error = %v", err)
	}
	if _, err := service.RemoveImage(ctx, "user_other", first.Meal.ID, first.Meal.Revision); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-user image delete error = %v", err)
	}
	if _, err := service.Delete(ctx, "user_other", first.Meal.ID, first.Meal.Revision); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-user meal delete error = %v", err)
	}
}

func mealTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	base := os.Getenv("TEST_DATABASE_URL")
	if base == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	admin, err := postgres.Open(context.Background(), base)
	if err != nil {
		t.Fatal(err)
	}
	database := "test_meals_" + strings.ToLower(rand.Text())
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
