SET search_path TO public;

-- Add media_id column to exercises table
ALTER TABLE exercises
ADD COLUMN media_id uuid REFERENCES media(id) ON DELETE SET NULL;

-- Add index for better query performance
CREATE INDEX idx_exercises_media_id ON exercises(media_id);
