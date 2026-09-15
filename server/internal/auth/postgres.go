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

func (s *PostgresStore) UserOwnsPhone(ctx context.Context, userID, phone string) (bool, error) {
	var owned bool
	err := s.pool.QueryRow(ctx, `select exists(select 1 from users where id=$1 and phone=$2 and deleted_at is null)`, userID, phone).Scan(&owned)
	return owned, err
}

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
	return sessionResult(userID, &input.Phone, expires, input.TermsVersion, input.PrivacyVersion, onboardingStatus(hasProfile, hasGoal)), nil
}

func (s *PostgresStore) Session(ctx context.Context, hash []byte, now time.Time) (SessionResult, error) {
	var userID, terms, privacy string
	var phone *string
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

func (s *PostgresStore) CreateWeChatBindingTicket(ctx context.Context, hash []byte, openID, unionID string, now, expires time.Time) error {
	var nullableUnionID any
	if unionID != "" {
		nullableUnionID = unionID
	}
	_, err := s.pool.Exec(ctx, `insert into wechat_binding_tickets(token_hash,openid,union_id,expires_at,created_at) values($1,$2,$3,$4,$5)`, hash, openID, nullableUnionID, expires, now)
	return err
}

func (s *PostgresStore) CreateSessionForWeChatIdentity(ctx context.Context, openID string, sessionHash []byte, now, expires time.Time) (SessionResult, bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return SessionResult{}, false, err
	}
	defer tx.Rollback(ctx)

	var userID, terms, privacy string
	var phone *string
	var hasProfile, hasGoal bool
	err = tx.QueryRow(ctx, `select u.id,u.phone,
		coalesce((select version from user_consents where user_id=u.id and kind='terms'),''),
		coalesce((select version from user_consents where user_id=u.id and kind='privacy'),''),
		exists(select 1 from profiles where user_id=u.id),
		exists(select 1 from goal_settings where user_id=u.id)
		from user_identities i join users u on u.id=i.user_id
		where i.provider='wechat_miniprogram' and i.provider_subject=$1 and u.deleted_at is null
		for update of i,u`, openID).Scan(&userID, &phone, &terms, &privacy, &hasProfile, &hasGoal)
	if errors.Is(err, pgx.ErrNoRows) {
		return SessionResult{}, false, nil
	}
	if err != nil {
		return SessionResult{}, false, err
	}
	if _, err := tx.Exec(ctx, `insert into sessions(id,user_id,token_hash,expires_at,client_type) values($1,$2,$3,$4,$5)`, "session_"+randText(), userID, sessionHash, expires, ClientWeChatMiniProgram); err != nil {
		return SessionResult{}, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return SessionResult{}, false, err
	}
	return sessionResult(userID, phone, expires, terms, privacy, onboardingStatus(hasProfile, hasGoal)), true, nil
}

func (s *PostgresStore) CreateWeChatUserAndSession(ctx context.Context, openID, unionID, termsVersion, privacyVersion string, sessionHash []byte, now, expires time.Time) (SessionResult, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return SessionResult{}, err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `select pg_advisory_xact_lock(hashtext($1))`, "wechat_miniprogram:"+openID); err != nil {
		return SessionResult{}, err
	}
	var userID string
	err = tx.QueryRow(ctx, `select i.user_id from user_identities i join users u on u.id=i.user_id where i.provider='wechat_miniprogram' and i.provider_subject=$1 and u.deleted_at is null`, openID).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		userID = "user_" + randText()
		if _, err = tx.Exec(ctx, `insert into users(id,phone) values($1,null)`, userID); err != nil {
			return SessionResult{}, err
		}
		var nullableUnionID any
		if unionID != "" {
			nullableUnionID = unionID
		}
		if _, err = tx.Exec(ctx, `insert into user_identities(id,user_id,provider,provider_subject,union_id,created_at,updated_at) values($1,$2,'wechat_miniprogram',$3,$4,$5,$5)`, "identity_"+randText(), userID, openID, nullableUnionID, now); err != nil {
			return SessionResult{}, err
		}
	} else if err != nil {
		return SessionResult{}, err
	}
	for _, consent := range []struct{ kind, version string }{{"terms", termsVersion}, {"privacy", privacyVersion}} {
		if _, err = tx.Exec(ctx, `insert into user_consents(user_id,kind,version,accepted_at) values($1,$2,$3,$4) on conflict(user_id,kind) do update set version=excluded.version,accepted_at=excluded.accepted_at`, userID, consent.kind, consent.version, now); err != nil {
			return SessionResult{}, err
		}
	}
	if _, err = tx.Exec(ctx, `insert into sessions(id,user_id,token_hash,expires_at,client_type) values($1,$2,$3,$4,$5)`, "session_"+randText(), userID, sessionHash, expires, ClientWeChatMiniProgram); err != nil {
		return SessionResult{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return SessionResult{}, err
	}
	return sessionResult(userID, nil, expires, termsVersion, privacyVersion, "profile_required"), nil
}

func (s *PostgresStore) BindWeChatPhoneAndCreateSession(ctx context.Context, ticketHash []byte, phone, termsVersion, privacyVersion string, sessionHash []byte, now, expires time.Time) (SessionResult, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return SessionResult{}, err
	}
	defer tx.Rollback(ctx)

	var openID string
	var unionID *string
	var ticketExpires time.Time
	var consumedAt *time.Time
	err = tx.QueryRow(ctx, `select openid,union_id,expires_at,consumed_at from wechat_binding_tickets where token_hash=$1 for update`, ticketHash).Scan(&openID, &unionID, &ticketExpires, &consumedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return SessionResult{}, &Error{Code: "WECHAT_BINDING_TICKET_INVALID", Message: "微信绑定凭证无效", Status: http.StatusUnprocessableEntity}
	}
	if err != nil {
		return SessionResult{}, err
	}
	if consumedAt != nil {
		return SessionResult{}, &Error{Code: "WECHAT_BINDING_TICKET_CONSUMED", Message: "微信绑定凭证已使用", Status: http.StatusConflict}
	}
	if !ticketExpires.After(now) {
		return SessionResult{}, &Error{Code: "WECHAT_BINDING_TICKET_EXPIRED", Message: "微信绑定凭证已过期", Status: http.StatusGone}
	}

	var identityUserID string
	var identityPhone *string
	identityErr := tx.QueryRow(ctx, `select i.user_id,u.phone from user_identities i join users u on u.id=i.user_id where i.provider='wechat_miniprogram' and i.provider_subject=$1 and u.deleted_at is null for update of i,u`, openID).Scan(&identityUserID, &identityPhone)
	if identityErr != nil && !errors.Is(identityErr, pgx.ErrNoRows) {
		return SessionResult{}, identityErr
	}
	identityExists := identityErr == nil

	var phoneUserID string
	phoneErr := tx.QueryRow(ctx, `select id from users where phone=$1 and deleted_at is null for update`, phone).Scan(&phoneUserID)
	if phoneErr != nil && !errors.Is(phoneErr, pgx.ErrNoRows) {
		return SessionResult{}, phoneErr
	}
	phoneExists := phoneErr == nil

	var userID string
	switch {
	case identityExists && identityPhone != nil && *identityPhone != phone:
		return SessionResult{}, identityConflict()
	case identityExists && phoneExists && identityUserID != phoneUserID:
		return SessionResult{}, identityConflict()
	case identityExists:
		userID = identityUserID
		if identityPhone == nil {
			if _, err := tx.Exec(ctx, `update users set phone=$2 where id=$1 and phone is null`, userID, phone); err != nil {
				return SessionResult{}, err
			}
		}
	case phoneExists:
		userID = phoneUserID
	default:
		userID = "user_" + randText()
		if _, err := tx.Exec(ctx, `insert into users(id,phone) values($1,$2)`, userID, phone); err != nil {
			return SessionResult{}, err
		}
	}

	if !identityExists {
		if _, err := tx.Exec(ctx, `insert into user_identities(id,user_id,provider,provider_subject,union_id,created_at,updated_at) values($1,$2,'wechat_miniprogram',$3,$4,$5,$5)`, "identity_"+randText(), userID, openID, unionID, now); err != nil {
			return SessionResult{}, err
		}
	}
	for _, consent := range []struct{ kind, version string }{{"terms", termsVersion}, {"privacy", privacyVersion}} {
		if _, err := tx.Exec(ctx, `insert into user_consents(user_id,kind,version,accepted_at) values($1,$2,$3,$4) on conflict(user_id,kind) do update set version=excluded.version,accepted_at=excluded.accepted_at`, userID, consent.kind, consent.version, now); err != nil {
			return SessionResult{}, err
		}
	}
	if _, err := tx.Exec(ctx, `insert into sessions(id,user_id,token_hash,expires_at,client_type) values($1,$2,$3,$4,$5)`, "session_"+randText(), userID, sessionHash, expires, ClientWeChatMiniProgram); err != nil {
		return SessionResult{}, err
	}
	if _, err := tx.Exec(ctx, `update wechat_binding_tickets set consumed_at=$2,user_id=$3 where token_hash=$1`, ticketHash, now, userID); err != nil {
		return SessionResult{}, err
	}

	var hasProfile, hasGoal bool
	if err := tx.QueryRow(ctx, `select exists(select 1 from profiles where user_id=$1),exists(select 1 from goal_settings where user_id=$1)`, userID).Scan(&hasProfile, &hasGoal); err != nil {
		return SessionResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return SessionResult{}, err
	}
	return sessionResult(userID, &phone, expires, termsVersion, privacyVersion, onboardingStatus(hasProfile, hasGoal)), nil
}

func invalidCode() error {
	return &Error{Code: "VERIFICATION_CODE_INVALID", Message: "验证码错误", Status: http.StatusUnprocessableEntity}
}

func identityConflict() error {
	return &Error{Code: "IDENTITY_CONFLICT", Message: "微信身份与手机号属于不同账户", Status: http.StatusConflict}
}
func randText() string { return rand.Text() }

func sessionResult(id string, phone *string, expires time.Time, terms, privacy, status string) SessionResult {
	var masked *string
	if phone != nil {
		value := (*phone)[:3] + " " + (*phone)[3:6] + "****" + (*phone)[10:]
		masked = &value
	}
	return SessionResult{Session: SessionView{ExpiresAt: expires}, User: UserView{ID: id, PhoneMasked: masked, OnboardingStatus: status}, Consents: ConsentsView{TermsVersion: terms, PrivacyVersion: privacy, CurrentAIImageProcessingVersion: CurrentAIImageProcessingVersion}}
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
