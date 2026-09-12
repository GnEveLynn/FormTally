package days

import (
	"context"
	"errors"
	"github.com/GnEveLynn/FormTally/server/internal/goals"
	"github.com/GnEveLynn/FormTally/server/internal/meals"
	"math"
	"time"
)

var ErrInvalidDate = errors.New("invalid or future date")

type Repository interface {
	Meals(context.Context, string, string) ([]meals.Meal, error)
	History(context.Context, string, string) ([]HistoryDay, error)
	Timezone(context.Context, string) (string, error)
}
type TargetProvider interface {
	Target(context.Context, string, string) (*goals.EffectiveTarget, error)
	CopyLatestTarget(context.Context, string, string, time.Time) (goals.EffectiveTarget, error)
}
type Service struct {
	repo    Repository
	targets TargetProvider
	now     func() time.Time
}

func NewService(repo Repository, targets TargetProvider) *Service {
	return &Service{repo: repo, targets: targets, now: time.Now}
}
func (s *Service) Get(ctx context.Context, userID, date string) (View, error) {
	parsed, err := time.Parse("2006-01-02", date)
	timezone, timezoneErr := s.repo.Timezone(ctx, userID)
	if timezoneErr != nil {
		return View{}, timezoneErr
	}
	location, locationErr := time.LoadLocation(timezone)
	if locationErr != nil {
		return View{}, locationErr
	}
	today := s.now().In(location).Format("2006-01-02")
	if err != nil || parsed.Format("2006-01-02") > today {
		return View{}, ErrInvalidDate
	}
	records, err := s.repo.Meals(ctx, userID, date)
	if err != nil {
		return View{}, err
	}
	target, err := s.targets.Target(ctx, userID, date)
	if err != nil {
		return View{}, err
	}
	if target == nil && (date == today || len(records) > 0) {
		copied, e := s.targets.CopyLatestTarget(ctx, userID, date, s.now())
		if e != nil {
			return View{}, e
		}
		target = &copied
	}
	view := View{LocalDate: date, MealGroups: []MealGroup{}}
	groups := map[string][]MealSummary{}
	for _, meal := range records {
		view.Totals.EnergyKcal += meal.Totals.EnergyKcal
		view.Totals.ProteinGrams += meal.Totals.ProteinGrams
		view.Totals.CarbGrams += meal.Totals.CarbGrams
		view.Totals.FatGrams += meal.Totals.FatGrams
		groups[meal.MealType] = append(groups[meal.MealType], MealSummary{ID: meal.ID, OccurredAt: meal.OccurredAt, Totals: meal.Totals})
	}
	view.Totals.ProteinGrams = round1(view.Totals.ProteinGrams)
	view.Totals.CarbGrams = round1(view.Totals.CarbGrams)
	view.Totals.FatGrams = round1(view.Totals.FatGrams)
	for _, kind := range []string{"breakfast", "lunch", "dinner", "snack"} {
		if list := groups[kind]; len(list) > 0 {
			view.MealGroups = append(view.MealGroups, MealGroup{MealType: kind, Meals: list})
		}
	}
	if target != nil {
		view.Target = &target.Target
		view.Progress = &Progress{Energy: metric(float64(view.Totals.EnergyKcal), float64(target.Target.EnergyKcal)), Protein: metric(view.Totals.ProteinGrams, float64(target.Target.ProteinGrams)), Carb: metric(view.Totals.CarbGrams, float64(target.Target.CarbGrams)), Fat: metric(view.Totals.FatGrams, float64(target.Target.FatGrams))}
	}
	return view, nil
}
func (s *Service) History(ctx context.Context, userID, month string) (HistoryView, error) {
	month = s.ResolveMonth(month)
	if _, err := time.Parse("2006-01", month); err != nil {
		return HistoryView{}, ErrInvalidDate
	}
	timezone, err := s.repo.Timezone(ctx, userID)
	if err != nil {
		return HistoryView{}, err
	}
	days, err := s.repo.History(ctx, userID, month)
	if err != nil {
		return HistoryView{}, err
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return HistoryView{}, err
	}
	today := s.now().In(location).Format("2006-01-02")
	visible := days[:0]
	for _, day := range days {
		if day.LocalDate <= today {
			visible = append(visible, day)
		}
	}
	return HistoryView{Month: month, Timezone: timezone, Days: visible}, nil
}

func (s *Service) ResolveMonth(month string) string {
	if month == "" {
		return s.now().Format("2006-01")
	}
	return month
}
func metric(consumed, target float64) MetricProgress {
	result := MetricProgress{Consumed: round1(consumed), Target: target, Remaining: round1(math.Max(target-consumed, 0)), OverBy: round1(math.Max(consumed-target, 0))}
	if target <= 0 {
		result.Status = "unavailable"
		return result
	}
	result.Percent = round1(consumed / target * 100)
	ratio := consumed / target
	if ratio < .9 {
		result.Status = "under"
	} else if ratio <= 1.1 {
		result.Status = "near"
	} else {
		result.Status = "over"
	}
	return result
}
func round1(value float64) float64 { return math.Round(value*10) / 10 }
