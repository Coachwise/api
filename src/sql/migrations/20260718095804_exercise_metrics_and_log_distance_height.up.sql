-- Exercises track different metrics: not everything is weight. Reps/duration
-- stay the per-set prescription (the sets XOR); these flags gate the extra
-- actuals an athlete logs per set. Weight defaults on to preserve today's UI.
ALTER TABLE exercises
    ADD COLUMN track_weight   boolean NOT NULL DEFAULT true,
    ADD COLUMN track_distance boolean NOT NULL DEFAULT false,
    ADD COLUMN track_grade    boolean NOT NULL DEFAULT false,
    ADD COLUMN track_height   boolean NOT NULL DEFAULT false;

-- Climbing exercises are graded and often about wall height/reach, not load.
UPDATE exercises SET track_grade = true, track_height = true, track_weight = false
WHERE sport_type = 'CLIMBING';

-- Cardio is usually distance-based rather than weighted.
UPDATE exercises SET track_distance = true, track_weight = false
WHERE sport_type = 'CARDIO';

-- Logged actuals: grade already exists; distance is metres, height is cm.
ALTER TABLE workout_logs
    ADD COLUMN distance numeric(10,2),
    ADD COLUMN height   numeric(10,2);

ALTER TABLE workout_logs
    ADD CONSTRAINT workout_logs_distance_check CHECK (distance IS NULL OR distance >= 0),
    ADD CONSTRAINT workout_logs_height_check   CHECK (height   IS NULL OR height   >= 0);
