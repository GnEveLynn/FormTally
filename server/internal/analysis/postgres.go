package analysis

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct{ pool *pgxpool.Pool }

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore { return &PostgresStore{pool: pool} }

func (s *PostgresStore) Insert(ctx context.Context, draft Draft) error {
	items, _ := json.Marshal(draft.Result.Items)
	warnings, _ := json.Marshal(draft.Result.Warning)
	_, err := s.pool.Exec(ctx, `insert into meal_analyses(id,user_id,image_key,image_width,image_height,occurred_at,local_date,meal_type,processing_mode,status,items,incomplete,warnings,expires_at,revision,created_at,updated_at) values($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`, draft.ID, draft.UserID, draft.ImageKey, draft.ImageWidth, draft.ImageHeight, draft.OccurredAt, draft.LocalDate, draft.MealType, draft.ProcessingMode, draft.Status, items, draft.Result.Incomplete, warnings, draft.ExpiresAt, draft.Revision, draft.CreatedAt, draft.UpdatedAt)
	return err
}

func (s *PostgresStore) Update(ctx context.Context, draft Draft) error {
	items, _ := json.Marshal(draft.Result.Items)
	warnings, _ := json.Marshal(draft.Result.Warning)
	failure, _ := json.Marshal(draft.Failure)
	if draft.Failure == nil {
		failure = nil
	}
	result, err := s.pool.Exec(ctx, `update meal_analyses set status=$3,items=$4,incomplete=$5,warnings=$6,failure=$7,model=$8,prompt_version=$9,response_status=$10,duration_ms=$11,meal_id=$12,revision=$13,updated_at=$14 where id=$1 and user_id=$2`, draft.ID, draft.UserID, draft.Status, items, draft.Result.Incomplete, warnings, failure, draft.Metadata.Model, draft.Metadata.PromptVersion, draft.Metadata.ResponseStatus, draft.Metadata.Duration.Milliseconds(), draft.MealID, draft.Revision, draft.UpdatedAt)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *PostgresStore) Get(ctx context.Context, userID, id string) (*Draft, error) {
	draft := &Draft{UserID: userID}
	var items, warnings, failure []byte
	var duration int64
	err := s.pool.QueryRow(ctx, `select id,image_key,image_width,image_height,occurred_at,local_date::text,meal_type,processing_mode,status,items,incomplete,warnings,failure,coalesce(model,''),coalesce(prompt_version,''),coalesce(response_status,''),coalesce(duration_ms,0),meal_id,expires_at,revision,created_at,updated_at from meal_analyses where id=$1 and user_id=$2`, id, userID).Scan(&draft.ID, &draft.ImageKey, &draft.ImageWidth, &draft.ImageHeight, &draft.OccurredAt, &draft.LocalDate, &draft.MealType, &draft.ProcessingMode, &draft.Status, &items, &draft.Result.Incomplete, &warnings, &failure, &draft.Metadata.Model, &draft.Metadata.PromptVersion, &draft.Metadata.ResponseStatus, &duration, &draft.MealID, &draft.ExpiresAt, &draft.Revision, &draft.CreatedAt, &draft.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	draft.Metadata.Duration = time.Duration(duration) * time.Millisecond
	if err := json.Unmarshal(items, &draft.Result.Items); err != nil {
		return nil, err
	}
	if len(warnings) > 0 && string(warnings) != "null" {
		if err := json.Unmarshal(warnings, &draft.Result.Warning); err != nil {
			return nil, err
		}
	}
	if len(failure) > 0 {
		draft.Failure = new(FailureView)
		if err := json.Unmarshal(failure, draft.Failure); err != nil {
			return nil, err
		}
	}
	return draft, nil
}

func (s *PostgresStore) Discard(ctx context.Context, userID, id string, revision int) (string, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	var key, status string
	var current int
	err = tx.QueryRow(ctx, `select image_key,status,revision from meal_analyses where id=$1 and user_id=$2 for update`, id, userID).Scan(&key, &status, &current)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if current != revision {
		return "", ErrRevisionConflict
	}
	if status != "review_required" && status != "failed" {
		return "", ErrStateConflict
	}
	if _, err = tx.Exec(ctx, `delete from meal_analyses where id=$1`, id); err != nil {
		return "", err
	}
	return key, tx.Commit(ctx)
}

func (s *PostgresStore) SaveConsent(ctx context.Context, userID, version string, now time.Time) error {
	_, err := s.pool.Exec(ctx, `insert into user_consents(user_id,kind,version,accepted_at) values($1,'ai_image_processing',$2,$3) on conflict(user_id,kind) do update set version=excluded.version,accepted_at=excluded.accepted_at`, userID, version, now)
	return err
}
