package profile

import (
	"context"
	"time"

	"github.com/GnEveLynn/FormTally/server/internal/goals"
)

type Service struct {
	store       *PostgresStore
	goalService *goals.Service
	now         func() time.Time
}

func NewService(store *PostgresStore, goalServices ...*goals.Service) *Service {
	service := &Service{store: store, now: time.Now}
	if len(goalServices) > 0 {
		service.goalService = goalServices[0]
	}
	return service
}

func (s *Service) Validate(input PutInput) error {
	eligibility := goals.EvaluateEligibility(goals.CalculationInput{
		BiologicalSex: input.BiologicalSex, BirthDate: input.BirthDate,
		HeightCm: input.HeightCm, WeightKg: input.WeightKg,
		ActivityLevel: input.ActivityLevel, Timezone: input.Timezone,
		At: s.now(), HealthContext: input.HealthContext,
	})
	for _, reason := range eligibility.Reasons {
		if reason != goals.ReasonHealthContextRequiresManual && reason != goals.ReasonAgeOutOfRange {
			return ErrValidation
		}
	}
	return nil
}

func (s *Service) Get(ctx context.Context, userID string) (*View, error) {
	view, err := s.store.Get(ctx, userID)
	if view != nil {
		s.applyEligibility(view)
	}
	return view, err
}

func (s *Service) Put(ctx context.Context, userID string, input PutInput) (View, error) {
	if err := s.Validate(input); err != nil {
		return View{}, err
	}
	view, err := s.store.Put(ctx, userID, input, s.now().UTC())
	if err != nil {
		return View{}, err
	}
	if s.goalService != nil {
		if err := s.goalService.RecalculateForProfile(ctx, userID); err != nil {
			return View{}, err
		}
	}
	s.applyEligibility(&view)
	return view, nil
}

func (s *Service) view(_ string, input PutInput, revision int, updatedAt time.Time) View {
	view := View{BiologicalSex: input.BiologicalSex, BirthDate: input.BirthDate, HeightCm: input.HeightCm, WeightKg: input.WeightKg, ActivityLevel: input.ActivityLevel, Timezone: input.Timezone, HealthContext: input.HealthContext, Revision: revision, UpdatedAt: updatedAt}
	s.applyEligibility(&view)
	return view
}

func (s *Service) applyEligibility(view *View) {
	view.AutomaticGoalEligible = goals.EvaluateEligibility(goals.CalculationInput{
		BiologicalSex: view.BiologicalSex, BirthDate: view.BirthDate,
		HeightCm: view.HeightCm, WeightKg: view.WeightKg,
		ActivityLevel: view.ActivityLevel, Timezone: view.Timezone,
		At: s.now(), HealthContext: view.HealthContext,
	}).Eligible
}
