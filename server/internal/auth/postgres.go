package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct{ pool *pgxpool.Pool }

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore { return &PostgresStore{pool: pool} }

func (s *PostgresStore) RetryAt(ctx context.Context, phone string, purpose Purpose) (time.Time, bool, error) {
	var retry time.Time
	err := s.pool.QueryRow(ctx, `select retry_after from login_codes where phone=$1 and purpose=$2 order by created_at desc limit 1`, phone, purpose).Scan(&retry)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, false, nil
	}
	return retry, err == nil, err
}

func (s *PostgresStore) InsertCode(ctx context.Context, id, phone string, purpose Purpose, hash []byte, expires, retry time.Time) error {
	_, err := s.pool.Exec(ctx, `insert into login_codes (id,phone,purpose,code_hash,expires_at,retry_after) values ($1,$2,$3,$4,$5,$6)`, id, phone, purpose, hash, expires, retry)
	return err
}

func (s *PostgresStore) ConsumeCodeAndCreateSession(ctx context.Context, input CreateSessionInput, hash, sessionHash []byte, now, expires time.Time) (SessionResult, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return SessionResult{}, err
	}
	defer tx.Rollback(ctx)
	var stored []byte
	var codeExpires time.Time
	var attempts, maxAttempts int
	var consumedAt *time.Time
	err = tx.QueryRow(ctx, `select code_hash,expires_at,attempts,max_attempts,consumed_at from login_codes where id=$1 and phone=$2 and purpose='login' for update`, input.VerificationRequestID, input.Phone).Scan(&stored, &codeExpires, &attempts, &maxAttempts, &consumedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return SessionResult{}, invalidCode()
	}
	if err != nil {
		return SessionResult{}, err
	}
	if consumedAt != nil || attempts >= maxAttempts || subtle.ConstantTimeCompare(stored, hash) != 1 {
		_, _ = tx.Exec(ctx, `update login_codes set attempts=attempts+1 where id=$1`, input.VerificationRequestID)
		_ = tx.Commit(ctx)
		return SessionResult{}, invalidCode()
	}
	if !codeExpires.After(now) {
		return SessionResult{}, &Error{Code: "VERIFICATION_CODE_EXPIRED", Message: "验证码已过期", Status: http.StatusGone}
	}
	userID := "user_" + randText()
	if err := tx.QueryRow(ctx, `insert into users (id,phone) values ($1,$2) on conflict(phone) do update set phone=excluded.phone returning id`, userID, input.Phone).Scan(&userID); err != nil {
		return SessionResult{}, err
	}
	for _, consent := range []struct{ kind, version string }{{"terms", input.TermsVersion}, {"privacy", input.PrivacyVersion}} {
		if _, err := tx.Exec(ctx, `insert into user_consents (user_id,kind,version,accepted_at) values ($1,$2,$3,$4) on conflict(user_id,kind) do update set version=excluded.version,accepted_at=excluded.accepted_at`, userID, consent.kind, consent.version, now); err != nil {
			return SessionResult{}, err
		}
	}
	if _, err := tx.Exec(ctx, `insert into sessions (id,user_id,token_hash,expires_at) values ($1,$2,$3,$4)`, "session_"+randText(), userID, sessionHash, expires); err != nil {
		return SessionResult{}, err
	}
	if _, err := tx.Exec(ctx, `update login_codes set consumed_at=$2 where id=$1`, input.VerificationRequestID, now); err != nil {
		return SessionResult{}, err
	}
	var hasProfile, hasGoal bool
	if err := tx.QueryRow(ctx, `select exists(select 1 from profiles where user_id=$1),exists(select 1 from goal_settings where user_id=$1)`, userID).Scan(&hasProfile, &hasGoal); err != nil {
		return SessionResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return SessionResult{}, err
	}
	return sessionResult(userID, input.Phone, expires, input.TermsVersion, input.PrivacyVersion, onboardingStatus(hasProfile, hasGoal)), nil
}

func (s *PostgresStore) Session(ctx context.Context, hash []byte, now time.Time) (SessionResult, error) {
	var userID, phone, terms, privacy string
	var expires time.Time
	var hasProfile, hasGoal bool
	err := s.pool.QueryRow(ctx, `select u.id,u.phone,s.expires_at,
		coalesce((select version from user_consents where user_id=u.id and kind='terms'),''),
		coalesce((select version from user_consents where user_id=u.id and kind='privacy'),''),
		exists(select 1 from profiles where user_id=u.id),
		exists(select 1 from goal_settings where user_id=u.id)
		from sessions s join users u on u.id=s.user_id where s.token_hash=$1 and s.revoked_at is null and s.expires_at>$2 and u.deleted_at is null`, hash, now).Scan(&userID, &phone, &expires, &terms, &privacy, &hasProfile, &hasGoal)
	if errors.Is(err, pgx.ErrNoRows) {
		return SessionResult{}, unauthenticated()
	}
	if err != nil {
		return SessionResult{}, err
	}
	return sessionResult(userID, phone, expires, terms, privacy, onboardingStatus(hasProfile, hasGoal)), nil
}

func (s *PostgresStore) RevokeSession(ctx context.Context, hash []byte, now time.Time) error {
	result, err := s.pool.Exec(ctx, `update sessions set revoked_at=$2 where token_hash=$1 and revoked_at is null`, hash, now)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return unauthenticated()
	}
	return nil
}

func invalidCode() error {
	return &Error{Code: "VERIFICATION_CODE_INVALID", Message: "验证码错误", Status: http.StatusUnprocessableEntity}
}
func randText() string { return rand.Text() }

func sessionResult(id, phone string, expires time.Time, terms, privacy, status string) SessionResult {
	return SessionResult{Session: SessionView{ExpiresAt: expires}, User: UserView{ID: id, PhoneMasked: phone[:3] + " " + phone[3:6] + "****" + phone[10:], OnboardingStatus: status}, Consents: ConsentsView{TermsVersion: terms, PrivacyVersion: privacy, CurrentAIImageProcessingVersion: CurrentAIImageProcessingVersion}}
}

func onboardingStatus(hasProfile, hasGoal bool) string {
	if !hasProfile {
		return "profile_required"
	}
	if !hasGoal {
		return "goal_required"
	}
	return "completed"
}
