ALTER TABLE public.test_requests RENAME COLUMN seen_at TO reviewed_at;
ALTER TABLE public.test_requests ADD COLUMN review_note text;
ALTER TABLE public.test_requests DROP CONSTRAINT test_requests_status_check;
UPDATE public.test_requests SET status = 'APPROVED' WHERE status = 'SEEN';
ALTER TABLE public.test_requests
    ADD CONSTRAINT test_requests_status_check CHECK (((status)::text = ANY ((ARRAY['PENDING'::character varying, 'SUBMITTED'::character varying, 'APPROVED'::character varying, 'REJECTED'::character varying])::text[])));
ALTER TABLE public.test_requests DROP COLUMN name;
DELETE FROM public.test_requests WHERE test_id IS NULL OR coach_id IS NULL;
ALTER TABLE public.test_requests
    ALTER COLUMN test_id SET NOT NULL,
    ALTER COLUMN coach_id SET NOT NULL;
