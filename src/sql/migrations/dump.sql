--
-- PostgreSQL database dump
--

-- Dumped from database version 14.4
-- Dumped by pg_dump version 14.4

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: uuid-ossp; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS "uuid-ossp" WITH SCHEMA public;


--
-- Name: EXTENSION "uuid-ossp"; Type: COMMENT; Schema: -; Owner: 
--

COMMENT ON EXTENSION "uuid-ossp" IS 'generate universally unique identifiers (UUIDs)';


--
-- Name: sports; Type: TYPE; Schema: public; Owner: coach-wise
--

CREATE TYPE public.sports AS ENUM (
    'FITNESS',
    'CLIMBING'
);


ALTER TYPE public.sports OWNER TO "coach-wise";

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: coaches; Type: TABLE; Schema: public; Owner: coach-wise
--

CREATE TABLE public.coaches (
    user_id uuid NOT NULL,
    specialties public.sports[] NOT NULL,
    created_at timestamp without time zone DEFAULT now(),
    updated_at timestamp without time zone DEFAULT now()
);


ALTER TABLE public.coaches OWNER TO "coach-wise";

--
-- Name: exercises; Type: TABLE; Schema: public; Owner: coach-wise
--

CREATE TABLE public.exercises (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    user_id uuid,
    name character varying(128),
    public boolean DEFAULT false,
    created_at timestamp without time zone DEFAULT now(),
    updated_at timestamp without time zone DEFAULT now()
);


ALTER TABLE public.exercises OWNER TO "coach-wise";

--
-- Name: plan_assignees; Type: TABLE; Schema: public; Owner: coach-wise
--

CREATE TABLE public.plan_assignees (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    plan_id uuid NOT NULL,
    user_id uuid NOT NULL,
    due_at timestamp without time zone,
    created_at timestamp without time zone DEFAULT now(),
    updated_at timestamp without time zone DEFAULT now()
);


ALTER TABLE public.plan_assignees OWNER TO "coach-wise";

--
-- Name: plan_exercises; Type: TABLE; Schema: public; Owner: coach-wise
--

CREATE TABLE public.plan_exercises (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    plan_id uuid NOT NULL,
    exercise_id uuid NOT NULL,
    exercise_order integer NOT NULL,
    rest_time interval NOT NULL
);


ALTER TABLE public.plan_exercises OWNER TO "coach-wise";

--
-- Name: plans; Type: TABLE; Schema: public; Owner: coach-wise
--

CREATE TABLE public.plans (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    user_id uuid NOT NULL,
    public boolean DEFAULT false,
    name character varying(128) NOT NULL,
    created_at timestamp without time zone DEFAULT now(),
    updated_at timestamp without time zone DEFAULT now()
);


ALTER TABLE public.plans OWNER TO "coach-wise";

--
-- Name: reps; Type: TABLE; Schema: public; Owner: coach-wise
--

CREATE TABLE public.reps (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    set_id uuid NOT NULL,
    rep_count integer,
    duration interval,
    rest_time interval NOT NULL,
    CONSTRAINT reps_check CHECK ((((rep_count IS NOT NULL) AND (duration IS NULL)) OR ((rep_count IS NULL) AND (duration IS NOT NULL)))),
    CONSTRAINT reps_rep_count_check CHECK ((rep_count > 0))
);


ALTER TABLE public.reps OWNER TO "coach-wise";

--
-- Name: schema_migrations; Type: TABLE; Schema: public; Owner: coach-wise
--

CREATE TABLE public.schema_migrations (
    version bigint NOT NULL,
    dirty boolean NOT NULL
);


ALTER TABLE public.schema_migrations OWNER TO "coach-wise";

--
-- Name: sets; Type: TABLE; Schema: public; Owner: coach-wise
--

CREATE TABLE public.sets (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    exercise_id uuid NOT NULL,
    set_number integer NOT NULL,
    rest_time interval NOT NULL,
    name character varying(128)
);


ALTER TABLE public.sets OWNER TO "coach-wise";

--
-- Name: users; Type: TABLE; Schema: public; Owner: coach-wise
--

CREATE TABLE public.users (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    username character varying(128) NOT NULL,
    first_name character varying(128),
    last_name character varying(128),
    email character varying(128) NOT NULL,
    phone character varying(128),
    created_at timestamp without time zone DEFAULT now(),
    updated_at timestamp without time zone DEFAULT now(),
    password text
);


ALTER TABLE public.users OWNER TO "coach-wise";

--
-- Data for Name: coaches; Type: TABLE DATA; Schema: public; Owner: coach-wise
--

COPY public.coaches (user_id, specialties, created_at, updated_at) FROM stdin;
\.


--
-- Data for Name: exercises; Type: TABLE DATA; Schema: public; Owner: coach-wise
--

COPY public.exercises (id, user_id, name, public, created_at, updated_at) FROM stdin;
\.


--
-- Data for Name: plan_assignees; Type: TABLE DATA; Schema: public; Owner: coach-wise
--

COPY public.plan_assignees (id, plan_id, user_id, due_at, created_at, updated_at) FROM stdin;
\.


--
-- Data for Name: plan_exercises; Type: TABLE DATA; Schema: public; Owner: coach-wise
--

COPY public.plan_exercises (id, plan_id, exercise_id, exercise_order, rest_time) FROM stdin;
\.


--
-- Data for Name: plans; Type: TABLE DATA; Schema: public; Owner: coach-wise
--

COPY public.plans (id, user_id, public, name, created_at, updated_at) FROM stdin;
\.


--
-- Data for Name: reps; Type: TABLE DATA; Schema: public; Owner: coach-wise
--

COPY public.reps (id, set_id, rep_count, duration, rest_time) FROM stdin;
\.


--
-- Data for Name: schema_migrations; Type: TABLE DATA; Schema: public; Owner: coach-wise
--

COPY public.schema_migrations (version, dirty) FROM stdin;
20240809154821  f
\.


--
-- Data for Name: sets; Type: TABLE DATA; Schema: public; Owner: coach-wise
--

COPY public.sets (id, exercise_id, set_number, rest_time, name) FROM stdin;
\.


--
-- Data for Name: users; Type: TABLE DATA; Schema: public; Owner: coach-wise
--

COPY public.users (id, username, first_name, last_name, email, phone, created_at, updated_at, password) FROM stdin;
effb5ee1-2d29-4225-a874-8bfd0890f062    jeyem   Ehsan   Mahmoudi        me@e-mahmoudi.me       \N      2024-08-10 18:52:00.772285      2024-08-10 18:52:00.772285      \N
\.


--
-- Name: coaches coaches_pkey; Type: CONSTRAINT; Schema: public; Owner: coach-wise
--

ALTER TABLE ONLY public.coaches
    ADD CONSTRAINT coaches_pkey PRIMARY KEY (user_id);


--
-- Name: exercises exercises_pkey; Type: CONSTRAINT; Schema: public; Owner: coach-wise
--

ALTER TABLE ONLY public.exercises
    ADD CONSTRAINT exercises_pkey PRIMARY KEY (id);


--
-- Name: plan_assignees plan_assignees_pkey; Type: CONSTRAINT; Schema: public; Owner: coach-wise
--

ALTER TABLE ONLY public.plan_assignees
    ADD CONSTRAINT plan_assignees_pkey PRIMARY KEY (id);


--
-- Name: plan_exercises plan_exercises_pkey; Type: CONSTRAINT; Schema: public; Owner: coach-wise
--

ALTER TABLE ONLY public.plan_exercises
    ADD CONSTRAINT plan_exercises_pkey PRIMARY KEY (id);


--
-- Name: plans plans_pkey; Type: CONSTRAINT; Schema: public; Owner: coach-wise
--

ALTER TABLE ONLY public.plans
    ADD CONSTRAINT plans_pkey PRIMARY KEY (id);


--
-- Name: reps reps_pkey; Type: CONSTRAINT; Schema: public; Owner: coach-wise
--

ALTER TABLE ONLY public.reps
    ADD CONSTRAINT reps_pkey PRIMARY KEY (id);


--
-- Name: schema_migrations schema_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: coach-wise
--

ALTER TABLE ONLY public.schema_migrations
    ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (version);


--
-- Name: sets sets_pkey; Type: CONSTRAINT; Schema: public; Owner: coach-wise
--

ALTER TABLE ONLY public.sets
    ADD CONSTRAINT sets_pkey PRIMARY KEY (id);


--
-- Name: users users_email_key; Type: CONSTRAINT; Schema: public; Owner: coach-wise
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_email_key UNIQUE (email);


--
-- Name: users users_phone_key; Type: CONSTRAINT; Schema: public; Owner: coach-wise
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_phone_key UNIQUE (phone);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: coach-wise
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: users users_username_key; Type: CONSTRAINT; Schema: public; Owner: coach-wise
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_username_key UNIQUE (username);


--
-- Name: plan_exercises fk_exercise; Type: FK CONSTRAINT; Schema: public; Owner: coach-wise
--

ALTER TABLE ONLY public.plan_exercises
    ADD CONSTRAINT fk_exercise FOREIGN KEY (exercise_id) REFERENCES public.exercises(id) ON DELETE CASCADE;


--
-- Name: plan_exercises fk_plan; Type: FK CONSTRAINT; Schema: public; Owner: coach-wise
--

ALTER TABLE ONLY public.plan_exercises
    ADD CONSTRAINT fk_plan FOREIGN KEY (plan_id) REFERENCES public.plans(id) ON DELETE CASCADE;


--
-- Name: plan_assignees fk_plan; Type: FK CONSTRAINT; Schema: public; Owner: coach-wise
--

ALTER TABLE ONLY public.plan_assignees
    ADD CONSTRAINT fk_plan FOREIGN KEY (plan_id) REFERENCES public.plans(id) ON DELETE CASCADE;


--
-- Name: reps fk_set; Type: FK CONSTRAINT; Schema: public; Owner: coach-wise
--

ALTER TABLE ONLY public.reps
    ADD CONSTRAINT fk_set FOREIGN KEY (set_id) REFERENCES public.sets(id) ON DELETE CASCADE;


--
-- Name: coaches fk_user; Type: FK CONSTRAINT; Schema: public; Owner: coach-wise
--

ALTER TABLE ONLY public.coaches
    ADD CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: exercises fk_user; Type: FK CONSTRAINT; Schema: public; Owner: coach-wise
--

ALTER TABLE ONLY public.exercises
    ADD CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: plans fk_user; Type: FK CONSTRAINT; Schema: public; Owner: coach-wise
--

ALTER TABLE ONLY public.plans
    ADD CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: plan_assignees fk_user; Type: FK CONSTRAINT; Schema: public; Owner: coach-wise
--

ALTER TABLE ONLY public.plan_assignees
    ADD CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: sets sets_exercise_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: coach-wise
--

ALTER TABLE ONLY public.sets
    ADD CONSTRAINT sets_exercise_id_fkey FOREIGN KEY (exercise_id) REFERENCES public.exercises(id) ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--
