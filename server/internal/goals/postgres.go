package goals

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct{ pool *pgxpool.Pool }

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore { return &PostgresStore{pool: pool} }

type ProfileRecord struct {
	BiologicalSex BiologicalSex
	BirthDate     string
	HeightCm      float64
	WeightKg      float64
	ActivityLevel ActivityLevel
	Timezone      string
	HealthContext HealthContext
}

func (s *PostgresStore) Profile(ctx context.Context, userID string) (*ProfileRecord, error) {
	profile := new(ProfileRecord)
	err := s.pool.QueryRow(ctx, `select biological_sex,birth_date::text,height_cm,weight_kg,activity_level,timezone,pregnant,breastfeeding,clinical_diet_required from profiles where user_id=$1`, userID).Scan(
		&profile.BiologicalSex, &profile.BirthDate, &profile.HeightCm, &profile.WeightKg, &profile.ActivityLevel, &profile.Timezone,
		&profile.HealthContext.Pregnant, &profile.HealthContext.Breastfeeding, &profile.HealthContext.ClinicalDietRequired,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return profile, err
}

func (s *PostgresStore) Settings(ctx context.Context, userID string) (*SettingsView, error) {
	settings := new(SettingsView)
	var objective *Objective
	var pace *Pace
	var energy, protein, carb, fat *int
	err := s.pool.QueryRow(ctx, `select mode,objective,pace,manual_energy_kcal,manual_protein_grams,manual_carb_grams,manual_fat_grams,revision,updated_at from goal_settings where user_id=$1`, userID).Scan(
		&settings.Mode, &objective, &pace, &energy, &protein, &carb, &fat, &settings.Revision, &settings.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if settings.Mode == ModeAutomatic {
		settings.Automatic = &AutomaticSettings{Objective: *objective}
		if pace != nil {
			settings.Automatic.Pace = *pace
		}
	} else {
		settings.Manual = &ManualSettings{Target: NutritionTarget{EnergyKcal: *energy, ProteinGrams: *protein, CarbGrams: *carb, FatGrams: *fat}}
	}
	return settings, nil
}

func (s *PostgresStore) Save(ctx context.Context, userID string, input SettingsInput, effective EffectiveTarget, now time.Time) (SaveResult, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return SaveResult{}, err
	}
	defer tx.Rollback(ctx)
	var current int
	err = tx.QueryRow(ctx, `select revision from goal_settings where user_id=$1 for update`, userID).Scan(&current)
	if errors.Is(err, pgx.ErrNoRows) {
		if input.ExpectedRevision != nil {
			return SaveResult{}, ErrRevisionConflict
		}
		current = 0
	} else if err != nil {
		return SaveResult{}, err
	} else if input.ExpectedRevision == nil || *input.ExpectedRevision != current {
		return SaveResult{}, ErrRevisionConflict
	}
	revision := current + 1
	var objective *Objective
	var pace *Pace
	var energy, protein, carb, fat *int
	if input.Mode == ModeAutomatic {
		objective = &input.Automatic.Objective
		if input.Automatic.Pace != PaceNone {
			pace = &input.Automatic.Pace
		}
	} else {
		target := input.Manual.Target
		energy, protein, carb, fat = &target.EnergyKcal, &target.ProteinGrams, &target.CarbGrams, &target.FatGrams
	}
	_, err = tx.Exec(ctx, `insert into goal_settings(user_id,mode,objective,pace,manual_energy_kcal,manual_protein_grams,manual_carb_grams,manual_fat_grams,revision,updated_at)
		values($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) on conflict(user_id) do update set mode=excluded.mode,objective=excluded.objective,pace=excluded.pace,manual_energy_kcal=excluded.manual_energy_kcal,manual_protein_grams=excluded.manual_protein_grams,manual_carb_grams=excluded.manual_carb_grams,manual_fat_grams=excluded.manual_fat_grams,revision=excluded.revision,updated_at=excluded.updated_at`,
		userID, input.Mode, objective, pace, energy, protein, carb, fat, revision, now)
	if err != nil {
		return SaveResult{}, err
	}
	if err := upsertTarget(ctx, tx, userID, effective.EffectiveFrom, effective, now); err != nil {
		return SaveResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return SaveResult{}, err
	}
	settings := SettingsView{Mode: input.Mode, Automatic: input.Automatic, Manual: input.Manual, Revision: revision, UpdatedAt: now}
	return SaveResult{Settings: settings, EffectiveTarget: effective}, nil
}

func upsertTarget(ctx context.Context, tx pgx.Tx, userID, date string, target EffectiveTarget, now time.Time) error {
	var calculation []byte
	var version *string
	if target.Calculation != nil {
		calculation, _ = json.Marshal(target.Calculation)
		version = &target.Calculation.CalculationVersion
	}
	warnings, _ := json.Marshal(target.Warnings)
	_, err := tx.Exec(ctx, `insert into daily_targets(id,user_id,local_date,energy_kcal,protein_grams,carb_grams,fat_grams,calculation_version,calculation,warnings,created_at)
		values($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) on conflict(user_id,local_date) do update set energy_kcal=excluded.energy_kcal,protein_grams=excluded.protein_grams,carb_grams=excluded.carb_grams,fat_grams=excluded.fat_grams,calculation_version=excluded.calculation_version,calculation=excluded.calculation,warnings=excluded.warnings,created_at=excluded.created_at`,
		"target_"+rand.Text(), userID, date, target.Target.EnergyKcal, target.Target.ProteinGrams, target.Target.CarbGrams, target.Target.FatGrams, version, calculation, warnings, now)
	return err
}

func (s *PostgresStore) Target(ctx context.Context, userID, date string) (*EffectiveTarget, error) {
	return scanTarget(s.pool.QueryRow(ctx, `select local_date::text,energy_kcal,protein_grams,carb_grams,fat_grams,calculation,warnings from daily_targets where user_id=$1 and local_date=$2`, userID, date))
}

func (s *PostgresStore) PendingTarget(ctx context.Context, userID, date string) (*EffectiveTarget, error) {
	return scanTarget(s.pool.QueryRow(ctx, `select local_date::text,energy_kcal,protein_grams,carb_grams,fat_grams,calculation,warnings from daily_targets where user_id=$1 and local_date>$2 order by local_date limit 1`, userID, date))
}

func scanTarget(row pgx.Row) (*EffectiveTarget, error) {
	target := new(EffectiveTarget)
	var calculation, warnings []byte
	err := row.Scan(&target.LocalDate, &target.Target.EnergyKcal, &target.Target.ProteinGrams, &target.Target.CarbGrams, &target.Target.FatGrams, &calculation, &warnings)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(calculation) > 0 {
		target.Calculation = new(GoalCalculation)
		if err := json.Unmarshal(calculation, target.Calculation); err != nil {
			return nil, err
		}
	}
	if err := json.Unmarshal(warnings, &target.Warnings); err != nil {
		return nil, err
	}
	return target, nil
}

func (s *PostgresStore) CopyLatestTarget(ctx context.Context, userID, date string, now time.Time) (EffectiveTarget, error) {
	target, err := scanTarget(s.pool.QueryRow(ctx, `select local_date::text,energy_kcal,protein_grams,carb_grams,fat_grams,calculation,warnings from daily_targets where user_id=$1 order by local_date desc limit 1`, userID))
	if err != nil || target == nil {
		return EffectiveTarget{}, err
	}
	target.LocalDate = date
	target.EffectiveFrom = ""
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return EffectiveTarget{}, err
	}
	defer tx.Rollback(ctx)
	if err := upsertTarget(ctx, tx, userID, date, *target, now); err != nil {
		return EffectiveTarget{}, err
	}
	return *target, tx.Commit(ctx)
}

func (s *PostgresStore) ReplaceTarget(ctx context.Context, userID string, target EffectiveTarget, now time.Time) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := upsertTarget(ctx, tx, userID, target.EffectiveFrom, target, now); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
