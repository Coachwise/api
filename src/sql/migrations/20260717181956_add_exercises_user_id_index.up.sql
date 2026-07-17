-- Exercise lists now filter on ownership (public = true OR user_id = viewer),
-- and user_id had no index.
CREATE INDEX IF NOT EXISTS idx_exercises_user_id ON exercises (user_id);
