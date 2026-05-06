-- First, normalize any quality values > 5 to fit within 1-5 scale
-- Map 1-10 to 1-5 by dividing by 2 and rounding up
UPDATE sessions
SET quality = CEIL(quality::numeric / 2.0)
WHERE quality > 5;

-- Now update the constraint
ALTER TABLE sessions
DROP CONSTRAINT IF EXISTS sessions_quality_check,
ADD CONSTRAINT sessions_quality_check CHECK (quality >= 1 AND quality <= 5);
