-- Coach applications: a user submits one, and it is approved/rejected via
-- unguessable capability links (decision_token) pushed to a Discord channel.
CREATE TABLE public.coach_applications (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    user_id uuid NOT NULL,
    full_name character varying(255) NOT NULL,
    specialty character varying(128) NOT NULL,
    experience_years integer NOT NULL DEFAULT 0,
    certifications text NOT NULL,
    bio text,
    website text,
    instagram text,
    status character varying(16) DEFAULT 'PENDING'::character varying NOT NULL,
    decision_token character varying(64) NOT NULL,
    review_note text,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    CONSTRAINT coach_applications_pkey PRIMARY KEY (id),
    CONSTRAINT coach_applications_status_check CHECK (((status)::text = ANY ((ARRAY['PENDING'::character varying, 'APPROVED'::character varying, 'REJECTED'::character varying])::text[]))),
    CONSTRAINT coach_applications_user_fk FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE
);

CREATE INDEX idx_coach_applications_user ON public.coach_applications (user_id, created_at DESC);
CREATE INDEX idx_coach_applications_status ON public.coach_applications (status);
