package profile

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct{ pool *pgxpool.Pool }

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore { return &PostgresStore{pool: pool} }

func (s *PostgresStore) Get(ctx context.Context, userID string) (*View, error) {
	view := new(View)
	err := s.pool.QueryRow(ctx, `select biological_sex,birth_date::text,height_cm,weight_kg,activity_level,timezone,pregnant,breastfeeding,clinical_diet_required,revision,updated_at from profiles where user_id=$1`, userID).Scan(
		&view.BiologicalSex, &view.BirthDate, &view.HeightCm, &view.WeightKg, &view.ActivityLevel, &view.Timezone,
		&view.HealthContext.Pregnant, &view.HealthContext.Breastfeeding, &view.HealthContext.ClinicalDietRequired, &view.Revision, &view.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return view, err
}

func (s *PostgresStore) Put(ctx context.Context, userID string, input PutInput, now time.Time) (View, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return View{}, err
	}
	defer tx.Rollback(ctx)
	var current int
	err = tx.QueryRow(ctx, `select revision from profiles where user_id=$1 for update`, userID).Scan(&current)
	if errors.Is(err, pgx.ErrNoRows) {
		if input.ExpectedRevision != nil {
			return View{}, ErrRevisionConflict
		}
		current = 0
	} else if err != nil {
		return View{}, err
	} else if input.ExpectedRevision == nil || *input.ExpectedRevision != current {
		return View{}, ErrRevisionConflict
	}
	revision := current + 1
	view := View{BiologicalSex: input.BiologicalSex, BirthDate: input.BirthDate, HeightCm: input.HeightCm, WeightKg: input.WeightKg, ActivityLevel: input.ActivityLevel, Timezone: input.Timezone, HealthContext: input.HealthContext, Revision: revision, UpdatedAt: now}
	_, err = tx.Exec(ctx, `insert into profiles(user_id,biological_sex,birth_date,height_cm,weight_kg,activity_level,timezone,pregnant,breastfeeding,clinical_diet_required,revision,updated_at)
		values($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		on conflict(user_id) do update set biological_sex=excluded.biological_sex,birth_date=excluded.birth_date,height_cm=excluded.height_cm,weight_kg=excluded.weight_kg,activity_level=excluded.activity_level,timezone=excluded.timezone,pregnant=excluded.pregnant,breastfeeding=excluded.breastfeeding,clinical_diet_required=excluded.clinical_diet_required,revision=excluded.revision,updated_at=excluded.updated_at`,
		userID, input.BiologicalSex, input.BirthDate, input.HeightCm, input.WeightKg, input.ActivityLevel, input.Timezone, input.HealthContext.Pregnant, input.HealthContext.Breastfeeding, input.HealthContext.ClinicalDietRequired, revision, now)
	if err != nil {
		return View{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return View{}, err
	}
	return view, nil
}
