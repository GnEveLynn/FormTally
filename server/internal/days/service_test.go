package days

import (
	"context"
	"testing"
	"time"

	"github.com/GnEveLynn/FormTally/server/internal/goals"
	"github.com/GnEveLynn/FormTally/server/internal/meals"
)

type fakeDays struct{ meals []meals.Meal }

func (f fakeDays) Meals(context.Context, string, string) ([]meals.Meal, error)   { return f.meals, nil }
func (f fakeDays) History(context.Context, string, string) ([]HistoryDay, error) { return nil, nil }

type fakeTargets struct{ target goals.EffectiveTarget }

func (f fakeTargets) Target(context.Context, string, string) (*goals.EffectiveTarget, error) {
	return &f.target, nil
}
func (f fakeTargets) CopyLatestTarget(context.Context, string, string, time.Time) (goals.EffectiveTarget, error) {
	return f.target, nil
}

func TestDaySummaryCountsOnlyPersistedMealsAndKeepsRealOverage(t *testing.T) {
	repo := fakeDays{meals: []meals.Meal{{ID: "meal_1", MealType: "lunch", Totals: meals.Nutrition{EnergyKcal: 2300, ProteinGrams: 140, CarbGrams: 250, FatGrams: 70}}}}
	targets := fakeTargets{target: goals.EffectiveTarget{LocalDate: "2026-09-11", Target: goals.NutritionTarget{EnergyKcal: 2000, ProteinGrams: 130, CarbGrams: 240, FatGrams: 60}}}
	service := NewService(repo, targets)
	service.now = func() time.Time { return time.Date(2026, 9, 11, 8, 0, 0, 0, time.UTC) }
	view, err := service.Get(context.Background(), "user", "2026-09-11")
	if err != nil {
		t.Fatal(err)
	}
	if view.Totals.EnergyKcal != 2300 || view.Progress.Energy.Status != "over" || view.Progress.Energy.OverBy != 300 || len(view.MealGroups) != 1 {
		t.Fatalf("view=%+v", view)
	}
}

func TestDaySummaryReturnsRealEmptyState(t *testing.T) {
	targets := fakeTargets{target: goals.EffectiveTarget{LocalDate: "2026-09-11", Target: goals.NutritionTarget{EnergyKcal: 2000, ProteinGrams: 130, CarbGrams: 240, FatGrams: 60}}}
	service := NewService(fakeDays{}, targets)
	service.now = time.Now
	view, err := service.Get(context.Background(), "user", "2026-09-11")
	if err != nil || view.Totals.EnergyKcal != 0 || len(view.MealGroups) != 0 {
		t.Fatalf("view=%+v err=%v", view, err)
	}
}
