SET search_path TO public;

-- Canonical, language-neutral identity for exercises. System/seeded exercises
-- carry a slug so seeding is idempotent and a single exercise isn't duplicated
-- per language. User-created exercises leave it NULL (Postgres allows many NULLs
-- under a UNIQUE constraint).
ALTER TABLE exercises ADD COLUMN slug varchar(96) UNIQUE;

-- Localized name/description: {"en": "...", "fa": "..."}. `name`/`description`
-- remain the default/fallback (and back `name` search). Display resolves as
-- name_i18n[lang] ?? name.
ALTER TABLE exercises ADD COLUMN name_i18n jsonb NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE exercises ADD COLUMN description_i18n jsonb NOT NULL DEFAULT '{}'::jsonb;

-- Categories live in their own table (not an enum) so they are extensible and
-- translatable, and can be scoped per sport as we add more. Same i18n JSONB
-- convention as exercises.
CREATE TABLE exercise_categories (
    id         uuid DEFAULT uuid_generate_v4() NOT NULL PRIMARY KEY,
    slug       varchar(96) NOT NULL UNIQUE,
    name_i18n  jsonb NOT NULL DEFAULT '{}'::jsonb,
    -- NULL = applies to any sport; otherwise scopes the category to one sport.
    sport_type exercise_sport_type,
    sort_order integer NOT NULL DEFAULT 0,
    created_at timestamp without time zone DEFAULT now(),
    updated_at timestamp without time zone DEFAULT now()
);

ALTER TABLE exercises
    ADD COLUMN category_id uuid REFERENCES exercise_categories(id) ON DELETE SET NULL;

CREATE INDEX idx_exercises_category_id ON exercises(category_id);
CREATE INDEX idx_exercise_categories_sport_type ON exercise_categories(sport_type);
