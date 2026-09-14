-- +goose Up
CREATE TABLE user_identities (
    id text PRIMARY KEY,
    user_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider text NOT NULL CHECK (provider IN ('wechat_miniprogram')),
    provider_subject text NOT NULL,
    union_id text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (provider, provider_subject)
);

CREATE TABLE wechat_binding_tickets (
    token_hash bytea PRIMARY KEY,
    openid text NOT NULL,
    union_id text,
    user_id text REFERENCES users(id) ON DELETE CASCADE,
    expires_at timestamptz NOT NULL,
    consumed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE sessions
    ADD COLUMN client_type text NOT NULL DEFAULT 'web'
    CHECK (client_type IN ('web', 'wechat_miniprogram'));

-- +goose Down
ALTER TABLE sessions DROP COLUMN client_type;
DROP TABLE wechat_binding_tickets;
DROP TABLE user_identities;
