ALTER TABLE public.plan_exercises
    DROP COLUMN IF EXISTS rounds,
    DROP COLUMN IF EXISTS round_rest,
    DROP COLUMN IF EXISTS round_duration;

DROP TABLE IF EXISTS public.exercise_items;

ALTER TABLE public.exercises
    DROP CONSTRAINT IF EXISTS exercises_kind_check;

ALTER TABLE public.exercises
    DROP COLUMN IF EXISTS kind,
    DROP COLUMN IF EXISTS rounds,
    DROP COLUMN IF EXISTS round_rest,
    DROP COLUMN IF EXISTS round_duration;
