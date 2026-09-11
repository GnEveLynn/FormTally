package profile

import (
	"context"
	"crypto/rand"
	"errors"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/GnEveLynn/FormTally/server/internal/goals"
	"github.com/GnEveLynn/FormTally/server/internal/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestProfileCreateUpdateRevisionAndOwnership(t *testing.T) {
	pool := profileTestPool(t)
	ctx := context.Background()
	for _, id := range []string{"user_profile_a", "user_profile_b"} {
		if _, err := pool.Exec(ctx, `insert into users(id,phone) values($1,$2)`, id, "+86138"+id[len(id)-1:]+"2345678"); err != nil {
			t.Fatal(err)
		}
	}
	service := NewService(NewPostgresStore(pool))
	service.now = func() time.Time { return time.Date(2026, 9, 11, 4, 0, 0, 0, time.UTC) }

	created, err := service.Put(ctx, "user_profile_a", validProfileInput())
	if err != nil {
		t.Fatal(err)
	}
	if created.Revision != 1 || !created.AutomaticGoalEligible {
		t.Fatalf("created profile = %+v", created)
	}
	other, err := service.Get(ctx, "user_profile_b")
	if err != nil || other != nil {
		t.Fatalf("other user's profile = %+v, %v", other, err)
	}

	revision := 1
	input := validProfileInput()
	input.WeightKg = 73
	input.ExpectedRevision = &revision
	updated, err := service.Put(ctx, "user_profile_a", input)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Revision != 2 || updated.WeightKg != 73 {
		t.Fatalf("updated profile = %+v", updated)
	}
	if _, err := service.Put(ctx, "user_profile_a", input); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("stale update err = %v", err)
	}
}

func TestProfileValidationAndAutomaticEligibility(t *testing.T) {
	service := NewService(nil)
	service.now = func() time.Time { return time.Date(2026, 9, 11, 4, 0, 0, 0, time.UTC) }
	for _, mutate := range []func(*PutInput){
		func(in *PutInput) { in.HeightCm = 99 },
		func(in *PutInput) { in.WeightKg = 351 },
		func(in *PutInput) { in.Timezone = "Mars/Olympus" },
	} {
		input := validProfileInput()
		mutate(&input)
		if err := service.Validate(input); !errors.Is(err, ErrValidation) {
			t.Fatalf("Validate(%+v) = %v", input, err)
		}
	}

	input := validProfileInput()
	input.HealthContext.Pregnant = true
	view := service.view("user", input, 1, service.now())
	if view.AutomaticGoalEligible {
		t.Fatalf("special health profile should require manual goal: %+v", view)
	}
}

func TestProfileUpdateRecalculatesPendingAutomaticTarget(t *testing.T) {
	pool := profileTestPool(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `insert into users(id,phone) values('user_profile_goal','+8613800000002')`); err != nil {
		t.Fatal(err)
	}
	profileService := NewService(NewPostgresStore(pool))
	profileService.now = func() time.Time { return time.Date(2026, 9, 11, 4, 0, 0, 0, time.UTC) }
	if _, err := profileService.Put(ctx, "user_profile_goal", validProfileInput()); err != nil {
		t.Fatal(err)
	}
	goalService := goals.NewService(goals.NewPostgresStore(pool), profileService.now)
	if _, err := goalService.Save(ctx, "user_profile_goal", goals.SettingsInput{Mode: goals.ModeAutomatic, Automatic: &goals.AutomaticSettings{Objective: goals.ObjectiveFatLoss, Pace: goals.PaceStandard}}); err != nil {
		t.Fatal(err)
	}

	profileService = NewService(NewPostgresStore(pool), goalService)
	profileService.now = func() time.Time { return time.Date(2026, 9, 11, 4, 0, 0, 0, time.UTC) }
	revision := 1
	input := validProfileInput()
	input.WeightKg = 80
	input.ExpectedRevision = &revision
	if _, err := profileService.Put(ctx, "user_profile_goal", input); err != nil {
		t.Fatal(err)
	}
	view, err := goalService.Get(ctx, "user_profile_goal")
	if err != nil {
		t.Fatal(err)
	}
	if view.PendingTarget == nil || view.PendingTarget.LocalDate != "2026-09-12" || view.PendingTarget.Target.EnergyKcal == 2220 {
		t.Fatalf("pending target was not recalculated: %+v", view.PendingTarget)
	}
}

func validProfileInput() PutInput {
	return PutInput{
		BiologicalSex: goals.SexMale,
		BirthDate:     "1995-06-18",
		HeightCm:      178,
		WeightKg:      72.5,
		ActivityLevel: goals.ActivityModerate,
		Timezone:      "Asia/Shanghai",
	}
}

func profileTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	base := os.Getenv("TEST_DATABASE_URL")
	if base == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	admin, err := postgres.Open(context.Background(), base)
	if err != nil {
		t.Fatal(err)
	}
	database := "test_profile_" + strings.ToLower(rand.Text())
	if _, err := admin.Exec(context.Background(), "create database "+pgx.Identifier{database}.Sanitize()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = admin.Exec(context.Background(), "drop database "+pgx.Identifier{database}.Sanitize()+" with (force)")
		admin.Close()
	})
	parsed, _ := url.Parse(base)
	parsed.Path = "/" + database
	if err := postgres.Migrate(context.Background(), parsed.String(), "../../migrations", "up"); err != nil {
		t.Fatal(err)
	}
	pool, err := postgres.Open(context.Background(), parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}
