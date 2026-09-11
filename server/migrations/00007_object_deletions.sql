-- +goose Up
CREATE TABLE object_deletions (
    id text PRIMARY KEY,
    object_key text NOT NULL UNIQUE,
    attempts integer NOT NULL DEFAULT 0,
    next_attempt_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL,
    completed_at timestamptz
);
CREATE INDEX object_deletions_due_idx ON object_deletions(next_attempt_at) WHERE completed_at IS NULL;

-- +goose Down
DROP TABLE object_deletions;
