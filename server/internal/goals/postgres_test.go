package goals

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

func TestGoalPreviewHasNoSideEffectsAndFirstSaveStartsToday(t *testing.T) {
	pool := goalsTestPool(t)
	ctx := context.Background()
	insertGoalTestUser(t, pool, "user_goals")
	service := NewService(NewPostgresStore(pool))
	service.now = goalTestNow

	preview, err := service.Preview(ctx, "user_goals", automaticInput(ObjectiveFatLoss, PaceStandard, nil))
	if err != nil {
		t.Fatal(err)
	}
	if preview.EffectiveFrom != "2026-09-11" || preview.Target.EnergyKcal != 2220 {
		t.Fatalf("preview = %+v", preview)
	}
	for _, table := range []string{"goal_settings", "daily_targets"} {
		var count int
		if err := pool.QueryRow(ctx, "select count(*) from "+table).Scan(&count); err != nil || count != 0 {
			t.Fatalf("%s count = %d, err = %v", table, count, err)
		}
	}

	saved, err := service.Save(ctx, "user_goals", automaticInput(ObjectiveFatLoss, PaceStandard, nil))
	if err != nil {
		t.Fatal(err)
	}
	if saved.Settings.Revision != 1 || saved.EffectiveTarget.EffectiveFrom != "2026-09-11" {
		t.Fatalf("saved = %+v", saved)
	}
	view, err := service.Get(ctx, "user_goals")
	if err != nil {
		t.Fatal(err)
	}
	if view.ActiveTarget == nil || view.ActiveTarget.Target.EnergyKcal != 2220 || view.PendingTarget != nil {
		t.Fatalf("goals view = %+v", view)
	}
}

func TestGoalUpdateStartsTomorrowAndKeepsActiveSnapshot(t *testing.T) {
	pool := goalsTestPool(t)
	ctx := context.Background()
	insertGoalTestUser(t, pool, "user_goal_update")
	service := NewService(NewPostgresStore(pool))
	service.now = goalTestNow
	if _, err := service.Save(ctx, "user_goal_update", automaticInput(ObjectiveFatLoss, PaceStandard, nil)); err != nil {
		t.Fatal(err)
	}
	revision := 1
	updated, err := service.Save(ctx, "user_goal_update", automaticInput(ObjectiveFatLoss, PaceFast, &revision))
	if err != nil {
		t.Fatal(err)
	}
	if updated.Settings.Revision != 2 || updated.EffectiveTarget.EffectiveFrom != "2026-09-12" || updated.EffectiveTarget.Target.EnergyKcal != 2090 {
		t.Fatalf("updated = %+v", updated)
	}
	view, err := service.Get(ctx, "user_goal_update")
	if err != nil {
		t.Fatal(err)
	}
	if view.ActiveTarget.Target.EnergyKcal != 2220 || view.PendingTarget.Target.EnergyKcal != 2090 {
		t.Fatalf("view = %+v", view)
	}
	if _, err := service.Save(ctx, "user_goal_update", automaticInput(ObjectiveFatLoss, PaceSlow, &revision)); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("stale save err = %v", err)
	}
}

func TestManualGoalWorksForSpecialHealthContextAndWarnsOnMacroMismatch(t *testing.T) {
	pool := goalsTestPool(t)
	ctx := context.Background()
	insertGoalTestUser(t, pool, "user_manual")
	if _, err := pool.Exec(ctx, `update profiles set pregnant=true where user_id='user_manual'`); err != nil {
		t.Fatal(err)
	}
	service := NewService(NewPostgresStore(pool))
	service.now = goalTestNow
	if _, err := service.Preview(ctx, "user_manual", automaticInput(ObjectiveMaintain, PaceNone, nil)); !errors.Is(err, ErrAutomaticGoalNotEligible) {
		t.Fatalf("automatic preview err = %v", err)
	}
	manual := SettingsInput{Mode: ModeManual, Manual: &ManualSettings{Target: NutritionTarget{EnergyKcal: 2100, ProteinGrams: 140, CarbGrams: 245, FatGrams: 62}}}
	preview, err := service.Preview(ctx, "user_manual", manual)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Calculation != nil || len(preview.Warnings) != 1 {
		t.Fatalf("manual preview = %+v", preview)
	}
}

func TestPastTargetIsCopiedOnceAndRemainsStable(t *testing.T) {
	pool := goalsTestPool(t)
	ctx := context.Background()
	insertGoalTestUser(t, pool, "user_past")
	service := NewService(NewPostgresStore(pool))
	service.now = goalTestNow
	if _, err := service.Save(ctx, "user_past", automaticInput(ObjectiveFatLoss, PaceStandard, nil)); err != nil {
		t.Fatal(err)
	}
	past, err := service.EnsureDailyTarget(ctx, "user_past", "2026-09-01")
	if err != nil || past.Target.EnergyKcal != 2220 {
		t.Fatalf("past target = %+v, %v", past, err)
	}
	revision := 1
	if _, err := service.Save(ctx, "user_past", automaticInput(ObjectiveMuscleGain, PaceFast, &revision)); err != nil {
		t.Fatal(err)
	}
	again, err := service.EnsureDailyTarget(ctx, "user_past", "2026-09-01")
	if err != nil || again.Target != past.Target {
		t.Fatalf("past target changed: before=%+v after=%+v err=%v", past, again, err)
	}
}

func automaticInput(objective Objective, pace Pace, revision *int) SettingsInput {
	return SettingsInput{Mode: ModeAutomatic, Automatic: &AutomaticSettings{Objective: objective, Pace: pace}, ExpectedRevision: revision}
}

func goalTestNow() time.Time { return time.Date(2026, 9, 11, 4, 0, 0, 0, time.UTC) }

func insertGoalTestUser(t *testing.T, pool *pgxpool.Pool, userID string) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `insert into users(id,phone) values($1,$2)`, userID, "+86"+rand.Text()); err != nil {
		t.Fatal(err)
	}
	_, err := pool.Exec(ctx, `insert into profiles(user_id,biological_sex,birth_date,height_cm,weight_kg,activity_level,timezone,revision,updated_at) values($1,'male','1995-06-18',178,72.5,'moderate','Asia/Shanghai',1,$2)`, userID, goalTestNow())
	if err != nil {
		t.Fatal(err)
	}
}

func goalsTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	base := os.Getenv("TEST_DATABASE_URL")
	if base == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	admin, err := postgres.Open(context.Background(), base)
	if err != nil {
		t.Fatal(err)
	}
	database := "test_goals_" + strings.ToLower(rand.Text())
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
