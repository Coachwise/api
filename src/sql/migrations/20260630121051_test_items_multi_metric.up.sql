-- A test item can measure a combination of metrics (e.g. weighted pull-up =
-- reps + weight), not just one. Replace the single `metric` with per-metric flags.
ALTER TABLE public.test_items
    ADD COLUMN track_reps boolean NOT NULL DEFAULT false,
    ADD COLUMN track_weight boolean NOT NULL DEFAULT false,
    ADD COLUMN track_time boolean NOT NULL DEFAULT false;

UPDATE public.test_items SET
    track_reps = (metric = 'COUNT'),
    track_weight = (metric = 'KG'),
    track_time = (metric = 'SECOND');

ALTER TABLE public.test_items DROP COLUMN metric;
