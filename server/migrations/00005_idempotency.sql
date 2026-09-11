-- +goose Up
CREATE TABLE idempotency_records (
    user_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    operation text NOT NULL,
    key text NOT NULL,
    request_hash text NOT NULL,
    state text NOT NULL CHECK (state IN ('processing','completed')),
    response_status integer,
    response_body jsonb,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    PRIMARY KEY(user_id, operation, key)
);
CREATE INDEX idempotency_expires_idx ON idempotency_records(expires_at);

-- +goose Down
DROP TABLE idempotency_records;
