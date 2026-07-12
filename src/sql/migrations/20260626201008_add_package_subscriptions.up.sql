-- Package subscriptions: the client relationship. A user becomes a coach's
-- "client" by enrolling in one of the coach's packages (coach-assigned or
-- athlete-subscribed). No payment yet — enrollment is free.
CREATE TABLE public.package_subscriptions (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    package_id uuid NOT NULL,
    coach_id uuid NOT NULL,
    client_id uuid NOT NULL,
    status character varying(16) DEFAULT 'ACTIVE'::character varying NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    CONSTRAINT package_subscriptions_pkey PRIMARY KEY (id),
    CONSTRAINT package_subscriptions_unique UNIQUE (package_id, client_id),
    CONSTRAINT package_subscriptions_status_check CHECK (((status)::text = ANY ((ARRAY['ACTIVE'::character varying, 'CANCELED'::character varying])::text[]))),
    CONSTRAINT package_subscriptions_package_fk FOREIGN KEY (package_id) REFERENCES public.coach_packages(id) ON DELETE CASCADE,
    CONSTRAINT package_subscriptions_coach_fk FOREIGN KEY (coach_id) REFERENCES public.users(id) ON DELETE CASCADE,
    CONSTRAINT package_subscriptions_client_fk FOREIGN KEY (client_id) REFERENCES public.users(id) ON DELETE CASCADE
);

CREATE INDEX idx_package_subscriptions_coach ON public.package_subscriptions (coach_id, status);
CREATE INDEX idx_package_subscriptions_client ON public.package_subscriptions (client_id, status);
