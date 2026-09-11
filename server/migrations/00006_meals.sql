-- +goose Up
CREATE TABLE meals (
    id text PRIMARY KEY,
    user_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    occurred_at timestamptz NOT NULL,
    local_date date NOT NULL,
    meal_type text NOT NULL CHECK (meal_type IN ('breakfast','lunch','dinner','snack')),
    image_key text,
    revision integer NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);
CREATE INDEX meals_user_date_idx ON meals(user_id,local_date,occurred_at);
CREATE TABLE meal_items (
    id text PRIMARY KEY,
    meal_id text NOT NULL REFERENCES meals(id) ON DELETE CASCADE,
    position integer NOT NULL,
    draft_item_id text,
    name text NOT NULL,
    grams numeric(7,1) NOT NULL,
    energy_kcal integer NOT NULL,
    protein_grams numeric(7,1) NOT NULL,
    carb_grams numeric(7,1) NOT NULL,
    fat_grams numeric(7,1) NOT NULL,
    basis_per_100_grams jsonb,
    origin text NOT NULL CHECK(origin IN ('ai','ai_modified','manual')),
    confidence text CHECK(confidence IN ('high','medium','low')),
    assumption text
);
ALTER TABLE meal_analyses ADD CONSTRAINT meal_analyses_meal_fk FOREIGN KEY(meal_id) REFERENCES meals(id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE meal_analyses DROP CONSTRAINT meal_analyses_meal_fk;
DROP TABLE meal_items;
DROP TABLE meals;
