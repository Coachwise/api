ALTER TABLE public.test_items ADD COLUMN metric public.units NOT NULL DEFAULT 'COUNT'::public.units;
UPDATE public.test_items SET metric = CASE
    WHEN track_weight THEN 'KG'::public.units
    WHEN track_time THEN 'SECOND'::public.units
    ELSE 'COUNT'::public.units END;
ALTER TABLE public.test_items
    DROP COLUMN track_reps,
    DROP COLUMN track_weight,
    DROP COLUMN track_time;
