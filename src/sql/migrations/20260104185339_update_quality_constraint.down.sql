ALTER TABLE sessions
DROP CONSTRAINT IF EXISTS sessions_quality_check,
ADD CONSTRAINT sessions_quality_check CHECK (quality >= 1 AND quality <= 10);
