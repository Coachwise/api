-- Connection (request/accept) system.

-- The request lifecycle. A row is kept forever once created (reject sets status,
-- never deletes); only the requester deletes it (cancel).
CREATE TABLE public.connection_requests (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    requester_id uuid NOT NULL,
    addressee_id uuid NOT NULL,
    status character varying(16) DEFAULT 'PENDING'::character varying NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    CONSTRAINT connection_requests_pkey PRIMARY KEY (id),
    CONSTRAINT connection_requests_unique UNIQUE (requester_id, addressee_id),
    CONSTRAINT connection_requests_no_self CHECK (requester_id <> addressee_id),
    CONSTRAINT connection_requests_status_check CHECK (((status)::text = ANY ((ARRAY['PENDING'::character varying, 'ACCEPTED'::character varying, 'REJECTED'::character varying])::text[]))),
    CONSTRAINT connection_requests_requester_fk FOREIGN KEY (requester_id) REFERENCES public.users(id) ON DELETE CASCADE,
    CONSTRAINT connection_requests_addressee_fk FOREIGN KEY (addressee_id) REFERENCES public.users(id) ON DELETE CASCADE
);

CREATE INDEX idx_connection_requests_addressee ON public.connection_requests (addressee_id, status);
CREATE INDEX idx_connection_requests_requester ON public.connection_requests (requester_id, status);

-- Established connections. One canonical row per pair (user1_id < user2_id by uuid).
-- Source of truth for "are these two connected".
CREATE TABLE public.connections (
    user1_id uuid NOT NULL,
    user2_id uuid NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    CONSTRAINT connections_pkey PRIMARY KEY (user1_id, user2_id),
    CONSTRAINT connections_order CHECK (user1_id < user2_id),
    CONSTRAINT connections_user1_fk FOREIGN KEY (user1_id) REFERENCES public.users(id) ON DELETE CASCADE,
    CONSTRAINT connections_user2_fk FOREIGN KEY (user2_id) REFERENCES public.users(id) ON DELETE CASCADE
);

CREATE INDEX idx_connections_user2 ON public.connections (user2_id);
