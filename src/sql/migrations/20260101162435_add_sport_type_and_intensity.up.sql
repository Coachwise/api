SET search_path TO public;

-- Create sport_type enum for exercises
CREATE TYPE exercise_sport_type AS ENUM ('STRENGTH', 'CLIMBING', 'CARDIO', 'MOBILITY', 'GENERAL');

-- Add sport_type to exercises table
ALTER TABLE exercises
ADD COLUMN sport_type exercise_sport_type DEFAULT 'GENERAL'::exercise_sport_type NOT NULL;

-- Add intensity to plan_exercises (1-10 scale, 1=min, 10=max)
ALTER TABLE plan_exercises
ADD COLUMN intensity integer CHECK (intensity >= 1 AND intensity <= 10) DEFAULT 5 NOT NULL;

-- Add indexes for better query performance
CREATE INDEX idx_exercises_sport_type ON exercises(sport_type);
