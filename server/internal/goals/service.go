package goals

import (
	"context"
	"time"
)

type Service struct {
	store *PostgresStore
	now   func() time.Time
}

func NewService(store *PostgresStore, clocks ...func() time.Time) *Service {
	clock := time.Now
	if len(clocks) > 0 {
		clock = clocks[0]
	}
	return &Service{store: store, now: clock}
}

func (s *Service) Preview(ctx context.Context, userID string, input SettingsInput) (EffectiveTarget, error) {
	profile, err := s.store.Profile(ctx, userID)
	if err != nil {
		return EffectiveTarget{}, err
	}
	if profile == nil {
		return EffectiveTarget{}, ErrProfileRequired
	}
	settings, err := s.store.Settings(ctx, userID)
	if err != nil {
		return EffectiveTarget{}, err
	}
	effectiveDate, err := effectiveDate(profile.Timezone, s.now(), settings != nil)
	if err != nil {
		return EffectiveTarget{}, ErrAutomaticGoalNotEligible
	}
	target, err := calculateSettings(*profile, input, s.now())
	if err != nil {
		return EffectiveTarget{}, err
	}
	target.EffectiveFrom = effectiveDate
	return target, nil
}

func (s *Service) Save(ctx context.Context, userID string, input SettingsInput) (SaveResult, error) {
	preview, err := s.Preview(ctx, userID, input)
	if err != nil {
		return SaveResult{}, err
	}
	return s.store.Save(ctx, userID, input, preview, s.now().UTC())
}

func (s *Service) Get(ctx context.Context, userID string) (View, error) {
	profile, err := s.store.Profile(ctx, userID)
	if err != nil || profile == nil {
		return View{}, err
	}
	settings, err := s.store.Settings(ctx, userID)
	if err != nil || settings == nil {
		return View{Settings: settings}, err
	}
	date, err := effectiveDate(profile.Timezone, s.now(), false)
	if err != nil {
		return View{}, err
	}
	if _, err := s.EnsureDailyTarget(ctx, userID, date); err != nil {
		return View{}, err
	}
	active, err := s.store.Target(ctx, userID, date)
	if err != nil {
		return View{}, err
	}
	pending, err := s.store.PendingTarget(ctx, userID, date)
	return View{Settings: settings, ActiveTarget: active, PendingTarget: pending}, err
}

func (s *Service) EnsureDailyTarget(ctx context.Context, userID, localDate string) (EffectiveTarget, error) {
	if existing, err := s.store.Target(ctx, userID, localDate); err != nil || existing != nil {
		if existing == nil {
			return EffectiveTarget{}, err
		}
		return *existing, err
	}
	return s.store.CopyLatestTarget(ctx, userID, localDate, s.now().UTC())
}

func (s *Service) RecalculateForProfile(ctx context.Context, userID string) error {
	settings, err := s.store.Settings(ctx, userID)
	if err != nil || settings == nil || settings.Mode != ModeAutomatic {
		return err
	}
	profile, err := s.store.Profile(ctx, userID)
	if err != nil || profile == nil {
		return err
	}
	input := SettingsInput{Mode: ModeAutomatic, Automatic: settings.Automatic}
	target, err := calculateSettings(*profile, input, s.now())
	if err != nil {
		return err
	}
	target.EffectiveFrom, err = effectiveDate(profile.Timezone, s.now(), true)
	if err != nil {
		return err
	}
	return s.store.ReplaceTarget(ctx, userID, target, s.now().UTC())
}

func calculateSettings(profile ProfileRecord, input SettingsInput, now time.Time) (EffectiveTarget, error) {
	switch input.Mode {
	case ModeAutomatic:
		if input.Automatic == nil || input.Manual != nil {
			return EffectiveTarget{}, ErrInvalidGoal
		}
		result, err := CalculateV2(CalculationInput{
			BiologicalSex: profile.BiologicalSex, BirthDate: profile.BirthDate,
			HeightCm: profile.HeightCm, WeightKg: profile.WeightKg,
			ActivityLevel: profile.ActivityLevel, Timezone: profile.Timezone,
			HealthContext: profile.HealthContext, At: now,
			Objective: input.Automatic.Objective, Pace: input.Automatic.Pace,
		})
		if err != nil {
			return EffectiveTarget{}, err
		}
		return EffectiveTarget{Target: result.Target, Calculation: &result.Calculation, Warnings: result.Warnings}, nil
	case ModeManual:
		if input.Manual == nil || input.Automatic != nil || !validManualTarget(input.Manual.Target) {
			return EffectiveTarget{}, ErrInvalidGoal
		}
		warnings := []string{}
		target := input.Manual.Target
		if target.ProteinGrams*4+target.CarbGrams*4+target.FatGrams*9 != target.EnergyKcal {
			warnings = append(warnings, "三大营养素换算热量与热量目标不一致")
		}
		return EffectiveTarget{Target: target, Warnings: warnings}, nil
	default:
		return EffectiveTarget{}, ErrInvalidGoal
	}
}

func validManualTarget(target NutritionTarget) bool {
	return target.EnergyKcal >= 1 && target.EnergyKcal <= 10000 &&
		target.ProteinGrams >= 1 && target.ProteinGrams <= 2000 &&
		target.CarbGrams >= 1 && target.CarbGrams <= 2000 &&
		target.FatGrams >= 1 && target.FatGrams <= 2000
}

func effectiveDate(timezone string, now time.Time, tomorrow bool) (string, error) {
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return "", err
	}
	date := now.In(location)
	if tomorrow {
		date = date.AddDate(0, 0, 1)
	}
	return date.Format("2006-01-02"), nil
}
