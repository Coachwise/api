ALTER TABLE sessions
ADD COLUMN intensity integer CHECK (intensity >= 1 AND intensity <= 10),
ADD COLUMN quality integer CHECK (quality >= 1 AND quality <= 10);
