-- A user-curated profile layout for their achievements (badges + records):
-- an ordered list of keys ("badge:<id>" / "record:<exercise_id>") plus a hidden
-- set. Applied on the profile trophy case.
CREATE TABLE public.profile_achievement_layouts (
    user_id uuid NOT NULL,
    layout jsonb NOT NULL DEFAULT '{"order":[],"hidden":[]}'::jsonb,
    updated_at timestamp without time zone NOT NULL DEFAULT now(),
    CONSTRAINT profile_achievement_layouts_pkey PRIMARY KEY (user_id),
    CONSTRAINT profile_achievement_layouts_user_fk FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE
);
