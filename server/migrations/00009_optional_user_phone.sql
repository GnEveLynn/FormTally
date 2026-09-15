-- +goose Up
ALTER TABLE users ALTER COLUMN phone DROP NOT NULL;

-- +goose Down
ALTER TABLE users ALTER COLUMN phone SET NOT NULL;
