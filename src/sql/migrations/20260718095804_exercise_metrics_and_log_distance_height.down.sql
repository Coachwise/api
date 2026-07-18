ALTER TABLE workout_logs
    DROP CONSTRAINT IF EXISTS workout_logs_distance_check,
    DROP CONSTRAINT IF EXISTS workout_logs_height_check;
ALTER TABLE workout_logs
    DROP COLUMN IF EXISTS distance,
    DROP COLUMN IF EXISTS height;
ALTER TABLE exercises
    DROP COLUMN IF EXISTS track_weight,
    DROP COLUMN IF EXISTS track_distance,
    DROP COLUMN IF EXISTS track_grade,
    DROP COLUMN IF EXISTS track_height;
