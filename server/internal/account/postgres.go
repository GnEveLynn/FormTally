package account

import (
	"context"
	"crypto/subtle"
	"errors"
	"time"

	"github.com/GnEveLynn/FormTally/server/internal/auth"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct{ pool *pgxpool.Pool }

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore { return &PostgresStore{pool: pool} }

func (s *PostgresStore) Delete(ctx context.Context, userID string, input DeleteInput, now time.Time) (Deletion, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Deletion{}, err
	}
	defer tx.Rollback(ctx)
	var stored []byte
	var expires time.Time
	var attempts, maxAttempts int
	var consumed *time.Time
	err = tx.QueryRow(ctx, `select c.code_hash,c.expires_at,c.attempts,c.max_attempts,c.consumed_at from users u join login_codes c on c.phone=u.phone where u.id=$1 and u.deleted_at is null and c.id=$2 and c.purpose='delete_account' for update of u,c`, userID, input.VerificationRequestID).Scan(&stored, &expires, &attempts, &maxAttempts, &consumed)
	if errors.Is(err, pgx.ErrNoRows) {
		return Deletion{}, ErrInvalidCode
	}
	if err != nil {
		return Deletion{}, err
	}
	if consumed != nil || attempts >= maxAttempts || subtle.ConstantTimeCompare(stored, auth.VerificationCodeHash(input.VerificationRequestID, input.Code)) != 1 {
		_, _ = tx.Exec(ctx, `update login_codes set attempts=attempts+1 where id=$1`, input.VerificationRequestID)
		_ = tx.Commit(ctx)
		return Deletion{}, ErrInvalidCode
	}
	if !expires.After(now) {
		return Deletion{}, ErrExpiredCode
	}
	_, err = tx.Exec(ctx, `insert into object_deletions(id,object_key,next_attempt_at,created_at) select 'deletion_'||encode(gen_random_bytes(16),'hex'),image_key,$2,$2 from (select image_key from meal_analyses where user_id=$1 union select image_key from meals where user_id=$1) images where image_key is not null on conflict(object_key) do nothing`, userID, now)
	if err != nil {
		return Deletion{}, err
	}
	if _, err = tx.Exec(ctx, `update login_codes set consumed_at=$2 where id=$1`, input.VerificationRequestID, now); err != nil {
		return Deletion{}, err
	}
	if _, err = tx.Exec(ctx, `delete from users where id=$1`, userID); err != nil {
		return Deletion{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Deletion{}, err
	}
	return Deletion{Status: "accepted", AccessRevokedAt: now, PurgeBy: now.Add(30 * 24 * time.Hour)}, nil
}
