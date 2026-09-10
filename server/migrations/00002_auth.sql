-- +goose Up
CREATE TABLE users (
    id text PRIMARY KEY,
    phone text NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
CREATE TABLE login_codes (
    id text PRIMARY KEY,
    phone text NOT NULL,
    purpose text NOT NULL CHECK (purpose IN ('login','delete_account')),
    code_hash bytea NOT NULL,
    expires_at timestamptz NOT NULL,
    retry_after timestamptz NOT NULL,
    attempts integer NOT NULL DEFAULT 0,
    max_attempts integer NOT NULL DEFAULT 5,
    consumed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX login_codes_phone_purpose_created_idx ON login_codes (phone, purpose, created_at DESC);
CREATE TABLE sessions (
    id text PRIMARY KEY,
    user_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash bytea NOT NULL UNIQUE,
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE user_consents (
    user_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind text NOT NULL,
    version text NOT NULL,
    accepted_at timestamptz NOT NULL,
    PRIMARY KEY (user_id, kind)
);

-- +goose Down
DROP TABLE user_consents;
DROP TABLE sessions;
DROP TABLE login_codes;
DROP TABLE users;
