SET search_path TO public;
DROP INDEX IF EXISTS idx_exercises_search_vector;
ALTER TABLE exercises DROP COLUMN IF EXISTS search_vector;
