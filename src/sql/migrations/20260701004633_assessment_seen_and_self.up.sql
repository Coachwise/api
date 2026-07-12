-- Lighten the assessment review and add athlete self-assessments.
-- 1) Review is now a coach "seen" acknowledgment (PENDING -> SUBMITTED -> SEEN),
--    not approve/reject. Records count toward PRs as soon as they're SUBMITTED.
-- 2) Self-assessments have no coach and no test template: an athlete picks
--    exercises and records results directly (test_id/coach_id NULL, own name).

ALTER TABLE public.test_requests
    ALTER COLUMN test_id DROP NOT NULL,
    ALTER COLUMN coach_id DROP NOT NULL,
    ADD COLUMN name character varying(128);

-- Status lifecycle: PENDING -> SUBMITTED -> SEEN.
ALTER TABLE public.test_requests DROP CONSTRAINT test_requests_status_check;
UPDATE public.test_requests SET status = 'SEEN' WHERE status = 'APPROVED';
UPDATE public.test_requests SET status = 'SUBMITTED' WHERE status = 'REJECTED';
ALTER TABLE public.test_requests
    ADD CONSTRAINT test_requests_status_check CHECK (((status)::text = ANY ((ARRAY['PENDING'::character varying, 'SUBMITTED'::character varying, 'SEEN'::character varying])::text[])));

-- reviewed_at/review_note collapse into a single seen acknowledgment.
ALTER TABLE public.test_requests DROP COLUMN review_note;
ALTER TABLE public.test_requests RENAME COLUMN reviewed_at TO seen_at;
