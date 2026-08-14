-- A group is an exercise that references other exercises and repeats them as
-- rounds (a circuit). Plans are unchanged: they still just hold exercises.

ALTER TABLE public.exercises
    ADD COLUMN IF NOT EXISTS kind varchar(16) NOT NULL DEFAULT 'SINGLE',
    -- Rounds for a group. round_duration set => run for that long (AMRAP)
    -- instead of a fixed count; the two are alternatives, not both.
    ADD COLUMN IF NOT EXISTS rounds integer,
    ADD COLUMN IF NOT EXISTS round_rest bigint NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS round_duration bigint;

ALTER TABLE public.exercises
    DROP CONSTRAINT IF EXISTS exercises_kind_check;
ALTER TABLE public.exercises
    ADD CONSTRAINT exercises_kind_check CHECK (kind IN ('SINGLE', 'GROUP'));

-- A group's children, in order, each with its own one-set prescription.
CREATE TABLE IF NOT EXISTS public.exercise_items (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    group_id uuid NOT NULL,
    exercise_id uuid NOT NULL,
    item_order integer NOT NULL,
    rep_count integer,
    duration bigint,
    rest_time bigint NOT NULL DEFAULT 0,
    created_at timestamp without time zone NOT NULL DEFAULT now(),
    updated_at timestamp without time zone NOT NULL DEFAULT now(),
    CONSTRAINT exercise_items_group_fk FOREIGN KEY (group_id)
        REFERENCES public.exercises(id) ON DELETE CASCADE,
    CONSTRAINT exercise_items_exercise_fk FOREIGN KEY (exercise_id)
        REFERENCES public.exercises(id) ON DELETE CASCADE,
    -- Same reps-xor-duration idiom as sets / plan_exercise_sets.
    CONSTRAINT exercise_items_reps_check CHECK (
        rep_count IS NOT NULL AND duration IS NULL
        OR rep_count IS NULL AND duration IS NOT NULL
    )
);

CREATE INDEX IF NOT EXISTS idx_exercise_items_group
    ON public.exercise_items (group_id, item_order);

-- Per-plan overrides of a group's rounds. NULL = inherit the group's own.
ALTER TABLE public.plan_exercises
    ADD COLUMN IF NOT EXISTS rounds integer,
    ADD COLUMN IF NOT EXISTS round_rest bigint,
    ADD COLUMN IF NOT EXISTS round_duration bigint;
