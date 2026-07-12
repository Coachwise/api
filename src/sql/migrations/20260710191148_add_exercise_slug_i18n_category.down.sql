SET search_path TO public;

DROP INDEX IF EXISTS idx_exercise_categories_sport_type;
DROP INDEX IF EXISTS idx_exercises_category_id;
ALTER TABLE exercises DROP COLUMN IF EXISTS category_id;
DROP TABLE IF EXISTS exercise_categories;

ALTER TABLE exercises DROP COLUMN IF EXISTS description_i18n;
ALTER TABLE exercises DROP COLUMN IF EXISTS name_i18n;
ALTER TABLE exercises DROP COLUMN IF EXISTS slug;
