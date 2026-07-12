-- Assessment tests: a coach builds a test (exercises + a units metric to measure
-- each), requests a client to take it, the athlete submits records, the coach
-- reviews/approves; approved records surface as PRs (+ coach-granted badges).
-- Reuses the existing units enum, exercises, and workout_logs (no redundant
-- type/table): a test submission is a workout_logs row (test_request_id set).

CREATE TABLE public.tests (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    coach_id uuid NOT NULL,
    name character varying(128) NOT NULL,
    description text,
    public boolean NOT NULL DEFAULT false,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    CONSTRAINT tests_pkey PRIMARY KEY (id),
    CONSTRAINT tests_coach_fk FOREIGN KEY (coach_id) REFERENCES public.users(id) ON DELETE CASCADE
);
CREATE INDEX idx_tests_coach ON public.tests (coach_id, created_at DESC);

CREATE TABLE public.test_items (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    test_id uuid NOT NULL,
    exercise_id uuid NOT NULL,
    metric public.units NOT NULL DEFAULT 'COUNT'::public.units,
    target_value numeric,
    item_order integer NOT NULL DEFAULT 0,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    CONSTRAINT test_items_pkey PRIMARY KEY (id),
    CONSTRAINT test_items_test_fk FOREIGN KEY (test_id) REFERENCES public.tests(id) ON DELETE CASCADE,
    CONSTRAINT test_items_exercise_fk FOREIGN KEY (exercise_id) REFERENCES public.exercises(id) ON DELETE CASCADE
);
CREATE INDEX idx_test_items_test ON public.test_items (test_id);

CREATE TABLE public.test_requests (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    test_id uuid NOT NULL,
    coach_id uuid NOT NULL,
    athlete_id uuid NOT NULL,
    status character varying(16) NOT NULL DEFAULT 'PENDING',
    note text,
    review_note text,
    submitted_at timestamp without time zone,
    reviewed_at timestamp without time zone,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    CONSTRAINT test_requests_pkey PRIMARY KEY (id),
    CONSTRAINT test_requests_status_check CHECK (((status)::text = ANY ((ARRAY['PENDING'::character varying, 'SUBMITTED'::character varying, 'APPROVED'::character varying, 'REJECTED'::character varying])::text[]))),
    CONSTRAINT test_requests_test_fk FOREIGN KEY (test_id) REFERENCES public.tests(id) ON DELETE CASCADE,
    CONSTRAINT test_requests_coach_fk FOREIGN KEY (coach_id) REFERENCES public.users(id) ON DELETE CASCADE,
    CONSTRAINT test_requests_athlete_fk FOREIGN KEY (athlete_id) REFERENCES public.users(id) ON DELETE CASCADE
);
CREATE INDEX idx_test_requests_athlete ON public.test_requests (athlete_id, status);
CREATE INDEX idx_test_requests_coach ON public.test_requests (coach_id, status);

CREATE TABLE public.achievements (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    athlete_id uuid NOT NULL,
    coach_id uuid NOT NULL,
    title character varying(128) NOT NULL,
    description text,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    CONSTRAINT achievements_pkey PRIMARY KEY (id),
    CONSTRAINT achievements_athlete_fk FOREIGN KEY (athlete_id) REFERENCES public.users(id) ON DELETE CASCADE,
    CONSTRAINT achievements_coach_fk FOREIGN KEY (coach_id) REFERENCES public.users(id) ON DELETE CASCADE
);
CREATE INDEX idx_achievements_athlete ON public.achievements (athlete_id, created_at DESC);

-- Reuse workout_logs for test submissions: a log belongs to a session OR a test.
ALTER TABLE public.workout_logs
    ADD COLUMN test_request_id uuid,
    ALTER COLUMN session_id DROP NOT NULL,
    ADD CONSTRAINT workout_logs_test_request_fk FOREIGN KEY (test_request_id) REFERENCES public.test_requests(id) ON DELETE CASCADE,
    ADD CONSTRAINT workout_logs_source_check CHECK ((session_id IS NOT NULL) OR (test_request_id IS NOT NULL));
CREATE INDEX idx_workout_logs_test_request ON public.workout_logs (test_request_id);
