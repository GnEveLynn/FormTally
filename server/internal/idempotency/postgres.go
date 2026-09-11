package idempotency

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct{ pool *pgxpool.Pool }

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore { return &PostgresStore{pool: pool} }

func (s *PostgresStore) Begin(ctx context.Context, userID, operation, key, requestHash string, now, expires time.Time) (Result, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Result{}, err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `insert into idempotency_records(user_id,operation,key,request_hash,state,expires_at,created_at,updated_at) values($1,$2,$3,$4,'processing',$5,$6,$6) on conflict do nothing`, userID, operation, key, requestHash, expires, now)
	if err != nil {
		return Result{}, err
	}
	if tag.RowsAffected() == 1 {
		if err := tx.Commit(ctx); err != nil {
			return Result{}, err
		}
		return Result{State: Started}, nil
	}
	var storedHash, state string
	var status *int
	var body []byte
	var storedExpiry time.Time
	if err := tx.QueryRow(ctx, `select request_hash,state,response_status,response_body,expires_at from idempotency_records where user_id=$1 and operation=$2 and key=$3 for update`, userID, operation, key).Scan(&storedHash, &state, &status, &body, &storedExpiry); err != nil {
		return Result{}, err
	}
	if storedHash != requestHash {
		return Result{}, ErrConflict
	}
	if !storedExpiry.After(now) {
		_, err = tx.Exec(ctx, `update idempotency_records set state='processing',response_status=null,response_body=null,expires_at=$4,updated_at=$5 where user_id=$1 and operation=$2 and key=$3`, userID, operation, key, expires, now)
		if err != nil {
			return Result{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return Result{}, err
		}
		return Result{State: Started}, nil
	}
	if state == "processing" {
		return Result{}, ErrInProgress
	}
	if status == nil {
		return Result{}, errors.New("completed idempotency record has no response")
	}
	return Result{State: Replayed, Status: *status, Body: body}, tx.Commit(ctx)
}

func (s *PostgresStore) Complete(ctx context.Context, userID, operation, key string, status int, body []byte, now time.Time) error {
	tag, err := s.pool.Exec(ctx, `update idempotency_records set state='completed',response_status=$4,response_body=$5,updated_at=$6 where user_id=$1 and operation=$2 and key=$3 and state='processing'`, userID, operation, key, status, body, now)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return pgx.ErrNoRows
	}
	return nil
}
