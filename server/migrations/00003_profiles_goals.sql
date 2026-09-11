-- +goose Up
CREATE TABLE profiles (
    user_id text PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    biological_sex text NOT NULL CHECK (biological_sex IN ('male','female')),
    birth_date date NOT NULL,
    height_cm numeric(5,1) NOT NULL,
    weight_kg numeric(5,1) NOT NULL,
    activity_level text NOT NULL CHECK (activity_level IN ('sedentary','light','moderate','high','very_high')),
    timezone text NOT NULL,
    pregnant boolean NOT NULL DEFAULT false,
    breastfeeding boolean NOT NULL DEFAULT false,
    clinical_diet_required boolean NOT NULL DEFAULT false,
    revision integer NOT NULL DEFAULT 1,
    updated_at timestamptz NOT NULL
);

CREATE TABLE goal_settings (
    user_id text PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    mode text NOT NULL CHECK (mode IN ('automatic','manual')),
    objective text CHECK (objective IN ('fat_loss','maintain','muscle_gain')),
    pace text CHECK (pace IN ('slow','standard','fast')),
    manual_energy_kcal integer,
    manual_protein_grams integer,
    manual_carb_grams integer,
    manual_fat_grams integer,
    revision integer NOT NULL DEFAULT 1,
    updated_at timestamptz NOT NULL
);

CREATE TABLE daily_targets (
    id text PRIMARY KEY,
    user_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    local_date date NOT NULL,
    energy_kcal integer NOT NULL,
    protein_grams integer NOT NULL,
    carb_grams integer NOT NULL,
    fat_grams integer NOT NULL,
    calculation_version text,
    calculation jsonb,
    warnings jsonb NOT NULL DEFAULT '[]'::jsonb,
    created_at timestamptz NOT NULL,
    UNIQUE (user_id, local_date)
);

-- +goose Down
DROP TABLE daily_targets;
DROP TABLE goal_settings;
DROP TABLE profiles;
