-- Per-plan exercise prescription. Mirrors `sets` (which stays on the exercise as
-- the reusable default/template), but keyed by plan_exercise so the same movement
-- can carry different sets/reps/rest in different plans without mutating the shared
-- exercise. Seeded from the exercise's default sets when an exercise is added.
CREATE TABLE public.plan_exercise_sets (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    plan_exercise_id uuid NOT NULL,
    name character varying(128),
    set_number integer NOT NULL,
    rest_time bigint NOT NULL DEFAULT 0,
    rep_count integer,
    duration bigint,
    created_at timestamp without time zone DEFAULT now(),
    updated_at timestamp without time zone DEFAULT now(),
    CONSTRAINT plan_exercise_sets_pkey PRIMARY KEY (id),
    CONSTRAINT pes_reps_check CHECK (
      ((rep_count IS NOT NULL AND duration IS NULL) OR (rep_count IS NULL AND duration IS NOT NULL))
    ),
    CONSTRAINT fk_plan_exercise FOREIGN KEY (plan_exercise_id)
      REFERENCES public.plan_exercises(id) ON DELETE CASCADE
);

CREATE INDEX idx_plan_exercise_sets_pe ON public.plan_exercise_sets(plan_exercise_id);
