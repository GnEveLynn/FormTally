-- +goose Up
CREATE TABLE meal_analyses (
    id text PRIMARY KEY,
    user_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    image_key text NOT NULL,
    image_width integer NOT NULL,
    image_height integer NOT NULL,
    occurred_at timestamptz NOT NULL,
    local_date date NOT NULL,
    meal_type text NOT NULL CHECK (meal_type IN ('breakfast','lunch','dinner','snack')),
    processing_mode text NOT NULL CHECK (processing_mode IN ('ai','manual')),
    status text NOT NULL CHECK (status IN ('processing','review_required','failed','saved')),
    items jsonb NOT NULL DEFAULT '[]'::jsonb,
    incomplete boolean NOT NULL DEFAULT false,
    warnings jsonb NOT NULL DEFAULT '[]'::jsonb,
    failure jsonb,
    model text,
    prompt_version text,
    response_status text,
    duration_ms bigint,
    meal_id text,
    expires_at timestamptz NOT NULL,
    revision integer NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);
CREATE INDEX meal_analyses_user_created_idx ON meal_analyses(user_id, created_at DESC);

-- +goose Down
DROP TABLE meal_analyses;
