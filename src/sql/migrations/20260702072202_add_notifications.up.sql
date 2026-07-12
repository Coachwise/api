-- In-app notifications: one row per recipient per event. Stores a type + small
-- structured `data` (not baked text) so the client renders localized, RTL copy.
CREATE TABLE public.notifications (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    user_id uuid NOT NULL,               -- recipient
    actor_id uuid,                       -- who triggered it (nullable)
    type character varying(48) NOT NULL,
    entity_type character varying(32),   -- for deep-linking (e.g. 'test', 'package')
    entity_id uuid,
    data jsonb NOT NULL DEFAULT '{}'::jsonb,
    read boolean NOT NULL DEFAULT false,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    CONSTRAINT notifications_pkey PRIMARY KEY (id),
    CONSTRAINT notifications_user_fk FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE,
    CONSTRAINT notifications_actor_fk FOREIGN KEY (actor_id) REFERENCES public.users(id) ON DELETE SET NULL
);
CREATE INDEX idx_notifications_user ON public.notifications (user_id, read, created_at DESC);
