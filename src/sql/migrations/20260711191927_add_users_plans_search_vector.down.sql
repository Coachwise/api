SET search_path TO public;
DROP INDEX IF EXISTS idx_plans_search_vector;
ALTER TABLE plans DROP COLUMN IF EXISTS search_vector;
DROP INDEX IF EXISTS idx_users_search_vector;
ALTER TABLE users DROP COLUMN IF EXISTS search_vector;
