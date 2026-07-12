ALTER TABLE public.workout_logs DROP CONSTRAINT IF EXISTS workout_logs_source_check;
ALTER TABLE public.workout_logs DROP CONSTRAINT IF EXISTS workout_logs_test_request_fk;
DROP INDEX IF EXISTS public.idx_workout_logs_test_request;
ALTER TABLE public.workout_logs DROP COLUMN IF EXISTS test_request_id;
DROP TABLE IF EXISTS public.achievements;
DROP TABLE IF EXISTS public.test_requests;
DROP TABLE IF EXISTS public.test_items;
DROP TABLE IF EXISTS public.tests;
