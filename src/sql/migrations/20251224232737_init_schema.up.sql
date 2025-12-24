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
-- Name: EXTENSION "uuid-ossp"; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION "uuid-ossp" IS 'generate universally unique identifiers (UUIDs)';


--
-- Name: otp_perposes; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.otp_perposes AS ENUM (
    'AUTH',
    'FORGET_PASSWORD'
);


--
-- Name: sides; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.sides AS ENUM (
    'LEFT',
    'RIGHT',
    'GENERAL'
);


--
-- Name: sports; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.sports AS ENUM (
    'FITNESS',
    'CLIMBING',
    'THERAPEUTIC'
);


--
-- Name: units; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.units AS ENUM (
    'KG',
    'CM',
    'SECOND',
    'COUNT'
);


--
-- Name: user_status; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.user_status AS ENUM (
    'ACTIVE',
    'INACTIVE',
    'SUSPENDED'
);


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: chat_members; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.chat_members (
    chat_id uuid NOT NULL,
    user_id uuid NOT NULL,
    role character varying(16) NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    CONSTRAINT chat_members_role_check CHECK (((role)::text = ANY ((ARRAY['OWNER'::character varying, 'ADMIN'::character varying, 'MEMBER'::character varying])::text[])))
);


--
-- Name: chats; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.chats (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    type character varying(16) NOT NULL,
    name text,
    owner_id uuid NOT NULL,
    chat_key text,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    CONSTRAINT chat_key_required_for_direct CHECK (((((type)::text = 'DIRECT'::text) AND (chat_key IS NOT NULL)) OR (((type)::text = 'CHANNEL'::text) AND (chat_key IS NULL)))),
    CONSTRAINT chats_type_check CHECK (((type)::text = ANY ((ARRAY['DIRECT'::character varying, 'CHANNEL'::character varying])::text[])))
);


--
-- Name: coaches; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.coaches (
    user_id uuid NOT NULL,
    specialties public.sports[] NOT NULL,
    created_at timestamp without time zone DEFAULT now(),
    updated_at timestamp without time zone DEFAULT now()
);


--
-- Name: exercises; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.exercises (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    user_id uuid,
    name character varying(128) NOT NULL,
    description text NOT NULL,
    public boolean DEFAULT false,
    created_at timestamp without time zone DEFAULT now(),
    updated_at timestamp without time zone DEFAULT now()
);


--
-- Name: feed_comments; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.feed_comments (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    feed_id uuid NOT NULL,
    user_id uuid NOT NULL,
    body text NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL
);


--
-- Name: feed_likes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.feed_likes (
    feed_id uuid NOT NULL,
    user_id uuid NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL
);


--
-- Name: feed_media; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.feed_media (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    feed_id uuid NOT NULL,
    kind character varying(16) NOT NULL,
    url text NOT NULL,
    thumbnail_url text,
    order_index integer DEFAULT 0 NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    CONSTRAINT feed_media_kind_check CHECK (((kind)::text = ANY ((ARRAY['IMAGE'::character varying, 'VIDEO'::character varying])::text[])))
);


--
-- Name: feed_tags; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.feed_tags (
    feed_id uuid NOT NULL,
    tag_id uuid NOT NULL
);


--
-- Name: feeds; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.feeds (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    body text,
    location text,
    visibility character varying(16) DEFAULT 'PUBLIC'::character varying NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    CONSTRAINT feeds_visibility_check CHECK (((visibility)::text = ANY ((ARRAY['PUBLIC'::character varying, 'PRIVATE'::character varying])::text[])))
);


--
-- Name: media; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.media (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    user_id uuid,
    url text NOT NULL,
    filename text NOT NULL,
    created_at timestamp without time zone DEFAULT now(),
    size_bytes bigint DEFAULT 0 NOT NULL
);


--
-- Name: messages; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.messages (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    sender_id uuid NOT NULL,
    body text NOT NULL,
    media_id uuid,
    read_at timestamp without time zone,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    chat_id uuid NOT NULL
);


--
-- Name: otps; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.otps (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    user_id uuid NOT NULL,
    code integer NOT NULL,
    created_at timestamp without time zone DEFAULT now(),
    expired_at timestamp with time zone DEFAULT (now() + '00:10:00'::interval) NOT NULL,
    is_verified boolean DEFAULT false NOT NULL,
    perpose public.otp_perposes DEFAULT 'AUTH'::public.otp_perposes NOT NULL,
    email text
);


--
-- Name: param_logs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.param_logs (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    value numeric(10,2) NOT NULL,
    note text,
    user_id uuid NOT NULL,
    param_id uuid NOT NULL,
    updated_at timestamp without time zone DEFAULT now(),
    created_at timestamp without time zone DEFAULT now()
);


--
-- Name: params; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.params (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    name text NOT NULL,
    description text,
    unit public.units DEFAULT 'COUNT'::public.units NOT NULL,
    side public.sides DEFAULT 'GENERAL'::public.sides NOT NULL,
    available_sports public.sports[] NOT NULL,
    updated_at timestamp without time zone DEFAULT now(),
    created_at timestamp without time zone DEFAULT now()
);


--
-- Name: plan_assignees; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.plan_assignees (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    plan_id uuid NOT NULL,
    user_id uuid NOT NULL,
    assigner uuid NOT NULL,
    created_at timestamp without time zone DEFAULT now()
);


--
-- Name: plan_exercises; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.plan_exercises (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    exercise_id uuid NOT NULL,
    plan_id uuid NOT NULL,
    exercise_order integer NOT NULL,
    rest_time bigint NOT NULL,
    created_at timestamp without time zone DEFAULT now()
);


--
-- Name: plans; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.plans (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    user_id uuid NOT NULL,
    public boolean DEFAULT false,
    name character varying(128) NOT NULL,
    created_at timestamp without time zone DEFAULT now(),
    updated_at timestamp without time zone DEFAULT now()
);


--
-- Name: sessions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sessions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    session_type character varying(20) NOT NULL,
    plan_id uuid,
    status character varying(20) DEFAULT 'ACTIVE'::character varying NOT NULL,
    started_at timestamp without time zone DEFAULT now() NOT NULL,
    ended_at timestamp without time zone,
    notes text,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    CONSTRAINT sessions_session_type_check CHECK (((session_type)::text = ANY ((ARRAY['PLAN'::character varying, 'FREESTYLE'::character varying, 'STRENGTH'::character varying, 'CLIMBING'::character varying])::text[]))),
    CONSTRAINT sessions_status_check CHECK (((status)::text = ANY ((ARRAY['ACTIVE'::character varying, 'COMPLETED'::character varying, 'ABANDONED'::character varying])::text[])))
);


--
-- Name: sets; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sets (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    name character varying(128),
    exercise_id uuid NOT NULL,
    set_number integer NOT NULL,
    rest_time bigint NOT NULL,
    rep_count integer,
    duration bigint,
    created_at timestamp without time zone DEFAULT now(),
    updated_at timestamp without time zone DEFAULT now(),
    CONSTRAINT reps_check CHECK ((((rep_count IS NOT NULL) AND (duration IS NULL)) OR ((rep_count IS NULL) AND (duration IS NOT NULL))))
);


--
-- Name: tags; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.tags (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name character varying(50) NOT NULL,
    normalized_name character varying(50) NOT NULL,
    is_system boolean DEFAULT false,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    search_vector tsvector GENERATED ALWAYS AS (to_tsvector('english'::regconfig, (COALESCE(name, ''::character varying))::text)) STORED
);


--
-- Name: tokens_blacklist; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.tokens_blacklist (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    token text,
    expired_at timestamp without time zone DEFAULT now()
);


--
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    username character varying(128) NOT NULL,
    password text,
    first_name character varying(128),
    last_name character varying(128),
    email character varying(128) NOT NULL,
    phone character varying(128),
    created_at timestamp without time zone DEFAULT now(),
    updated_at timestamp without time zone DEFAULT now(),
    status public.user_status DEFAULT 'INACTIVE'::public.user_status NOT NULL,
    password_expired boolean DEFAULT false NOT NULL,
    job_title text,
    bio text,
    pro_until timestamp without time zone,
    avatar_id uuid
);


--
-- Name: workout_logs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.workout_logs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    session_id uuid NOT NULL,
    exercise_id uuid,
    exercise_name character varying(255),
    set_number integer NOT NULL,
    reps integer,
    weight numeric(10,2),
    rpe numeric(3,1),
    duration_seconds integer,
    grade character varying(10),
    completed boolean DEFAULT false,
    attempts integer,
    notes text,
    logged_at timestamp without time zone DEFAULT now() NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    CONSTRAINT workout_logs_attempts_check CHECK ((attempts > 0)),
    CONSTRAINT workout_logs_duration_seconds_check CHECK ((duration_seconds > 0)),
    CONSTRAINT workout_logs_reps_check CHECK ((reps > 0)),
    CONSTRAINT workout_logs_rpe_check CHECK (((rpe >= (0)::numeric) AND (rpe <= (10)::numeric))),
    CONSTRAINT workout_logs_set_number_check CHECK ((set_number > 0)),
    CONSTRAINT workout_logs_weight_check CHECK ((weight >= (0)::numeric))
);


--
-- Name: workout_logs_tags; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.workout_logs_tags (
    workout_log_id uuid NOT NULL,
    tag_id uuid NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL
);


--
-- Name: chat_members chat_members_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.chat_members
    ADD CONSTRAINT chat_members_pkey PRIMARY KEY (chat_id, user_id);


--
-- Name: chats chats_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.chats
    ADD CONSTRAINT chats_pkey PRIMARY KEY (id);


--
-- Name: coaches coaches_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.coaches
    ADD CONSTRAINT coaches_pkey PRIMARY KEY (user_id);


--
-- Name: exercises exercises_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.exercises
    ADD CONSTRAINT exercises_pkey PRIMARY KEY (id);


--
-- Name: feed_comments feed_comments_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.feed_comments
    ADD CONSTRAINT feed_comments_pkey PRIMARY KEY (id);


--
-- Name: feed_likes feed_likes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.feed_likes
    ADD CONSTRAINT feed_likes_pkey PRIMARY KEY (feed_id, user_id);


--
-- Name: feed_media feed_media_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.feed_media
    ADD CONSTRAINT feed_media_pkey PRIMARY KEY (id);


--
-- Name: feed_tags feed_tags_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.feed_tags
    ADD CONSTRAINT feed_tags_pkey PRIMARY KEY (feed_id, tag_id);


--
-- Name: feeds feeds_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.feeds
    ADD CONSTRAINT feeds_pkey PRIMARY KEY (id);


--
-- Name: media media_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.media
    ADD CONSTRAINT media_pkey PRIMARY KEY (id);


--
-- Name: media media_url_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.media
    ADD CONSTRAINT media_url_key UNIQUE (url);


--
-- Name: messages messages_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.messages
    ADD CONSTRAINT messages_pkey PRIMARY KEY (id);


--
-- Name: otps otps_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.otps
    ADD CONSTRAINT otps_pkey PRIMARY KEY (id);


--
-- Name: param_logs param_logs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.param_logs
    ADD CONSTRAINT param_logs_pkey PRIMARY KEY (id);


--
-- Name: params params_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.params
    ADD CONSTRAINT params_name_key UNIQUE (name);


--
-- Name: params params_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.params
    ADD CONSTRAINT params_pkey PRIMARY KEY (id);


--
-- Name: plan_assignees plan_assignees_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plan_assignees
    ADD CONSTRAINT plan_assignees_pkey PRIMARY KEY (id);


--
-- Name: plan_exercises plan_exercises_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plan_exercises
    ADD CONSTRAINT plan_exercises_pkey PRIMARY KEY (id);


--
-- Name: plans plans_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plans
    ADD CONSTRAINT plans_pkey PRIMARY KEY (id);



--
-- Name: sessions sessions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sessions
    ADD CONSTRAINT sessions_pkey PRIMARY KEY (id);


--
-- Name: sets sets_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sets
    ADD CONSTRAINT sets_pkey PRIMARY KEY (id);


--
-- Name: tags tags_normalized_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tags
    ADD CONSTRAINT tags_normalized_name_key UNIQUE (normalized_name);


--
-- Name: tags tags_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tags
    ADD CONSTRAINT tags_pkey PRIMARY KEY (id);


--
-- Name: tokens_blacklist tokens_blacklist_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tokens_blacklist
    ADD CONSTRAINT tokens_blacklist_pkey PRIMARY KEY (id);


--
-- Name: tokens_blacklist tokens_blacklist_token_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tokens_blacklist
    ADD CONSTRAINT tokens_blacklist_token_key UNIQUE (token);


--
-- Name: chats uniq_direct_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.chats
    ADD CONSTRAINT uniq_direct_key UNIQUE (chat_key);


--
-- Name: users users_email_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_email_key UNIQUE (email);


--
-- Name: users users_phone_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_phone_key UNIQUE (phone);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: users users_username_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_username_key UNIQUE (username);


--
-- Name: workout_logs workout_logs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workout_logs
    ADD CONSTRAINT workout_logs_pkey PRIMARY KEY (id);


--
-- Name: workout_logs_tags workout_logs_tags_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workout_logs_tags
    ADD CONSTRAINT workout_logs_tags_pkey PRIMARY KEY (workout_log_id, tag_id);


--
-- Name: idx_feed_comments_feed_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_feed_comments_feed_id ON public.feed_comments USING btree (feed_id);


--
-- Name: idx_feed_media_feed_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_feed_media_feed_id ON public.feed_media USING btree (feed_id);


--
-- Name: idx_feeds_user_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_feeds_user_created_at ON public.feeds USING btree (user_id, created_at DESC);


--
-- Name: idx_feeds_visibility_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_feeds_visibility_created_at ON public.feeds USING btree (visibility, created_at DESC);


--
-- Name: idx_messages_chat_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_messages_chat_created_at ON public.messages USING btree (chat_id, created_at DESC);


--
-- Name: idx_sessions_started_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sessions_started_at ON public.sessions USING btree (started_at);


--
-- Name: idx_sessions_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sessions_status ON public.sessions USING btree (status);


--
-- Name: idx_sessions_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sessions_user_id ON public.sessions USING btree (user_id);


--
-- Name: idx_tags_is_system; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_tags_is_system ON public.tags USING btree (is_system);


--
-- Name: idx_tags_normalized_name; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_tags_normalized_name ON public.tags USING btree (normalized_name);


--
-- Name: idx_tags_search_vector; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_tags_search_vector ON public.tags USING gin (search_vector);


--
-- Name: idx_workout_logs_exercise_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workout_logs_exercise_id ON public.workout_logs USING btree (exercise_id);


--
-- Name: idx_workout_logs_logged_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workout_logs_logged_at ON public.workout_logs USING btree (logged_at);


--
-- Name: idx_workout_logs_session_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workout_logs_session_id ON public.workout_logs USING btree (session_id);


--
-- Name: idx_workout_logs_tags_tag_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workout_logs_tags_tag_id ON public.workout_logs_tags USING btree (tag_id);


--
-- Name: idx_workout_logs_tags_workout_log_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workout_logs_tags_workout_log_id ON public.workout_logs_tags USING btree (workout_log_id);


--
-- Name: chat_members chat_members_chat_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.chat_members
    ADD CONSTRAINT chat_members_chat_id_fkey FOREIGN KEY (chat_id) REFERENCES public.chats(id) ON DELETE CASCADE;


--
-- Name: chat_members chat_members_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.chat_members
    ADD CONSTRAINT chat_members_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: chats chats_owner_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.chats
    ADD CONSTRAINT chats_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: feed_comments feed_comments_feed_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.feed_comments
    ADD CONSTRAINT feed_comments_feed_id_fkey FOREIGN KEY (feed_id) REFERENCES public.feeds(id) ON DELETE CASCADE;


--
-- Name: feed_comments feed_comments_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.feed_comments
    ADD CONSTRAINT feed_comments_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: feed_likes feed_likes_feed_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.feed_likes
    ADD CONSTRAINT feed_likes_feed_id_fkey FOREIGN KEY (feed_id) REFERENCES public.feeds(id) ON DELETE CASCADE;


--
-- Name: feed_likes feed_likes_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.feed_likes
    ADD CONSTRAINT feed_likes_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: feed_media feed_media_feed_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.feed_media
    ADD CONSTRAINT feed_media_feed_id_fkey FOREIGN KEY (feed_id) REFERENCES public.feeds(id) ON DELETE CASCADE;


--
-- Name: feed_tags feed_tags_feed_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.feed_tags
    ADD CONSTRAINT feed_tags_feed_id_fkey FOREIGN KEY (feed_id) REFERENCES public.feeds(id) ON DELETE CASCADE;


--
-- Name: feed_tags feed_tags_tag_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.feed_tags
    ADD CONSTRAINT feed_tags_tag_id_fkey FOREIGN KEY (tag_id) REFERENCES public.tags(id) ON DELETE CASCADE;


--
-- Name: feeds feeds_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.feeds
    ADD CONSTRAINT feeds_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: plan_assignees fk_assigner; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plan_assignees
    ADD CONSTRAINT fk_assigner FOREIGN KEY (assigner) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: plan_exercises fk_exercise; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plan_exercises
    ADD CONSTRAINT fk_exercise FOREIGN KEY (exercise_id) REFERENCES public.exercises(id) ON DELETE CASCADE;


--
-- Name: sets fk_exercise; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sets
    ADD CONSTRAINT fk_exercise FOREIGN KEY (exercise_id) REFERENCES public.exercises(id) ON DELETE CASCADE;


--
-- Name: messages fk_messages_chat; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.messages
    ADD CONSTRAINT fk_messages_chat FOREIGN KEY (chat_id) REFERENCES public.chats(id) ON DELETE CASCADE;


--
-- Name: param_logs fk_param; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.param_logs
    ADD CONSTRAINT fk_param FOREIGN KEY (param_id) REFERENCES public.params(id) ON DELETE CASCADE;


--
-- Name: plan_assignees fk_plan; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plan_assignees
    ADD CONSTRAINT fk_plan FOREIGN KEY (plan_id) REFERENCES public.plans(id) ON DELETE CASCADE;


--
-- Name: plan_exercises fk_plan; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plan_exercises
    ADD CONSTRAINT fk_plan FOREIGN KEY (plan_id) REFERENCES public.plans(id) ON DELETE CASCADE;


--
-- Name: coaches fk_user; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.coaches
    ADD CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: exercises fk_user; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.exercises
    ADD CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: plans fk_user; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plans
    ADD CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: plan_assignees fk_user; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plan_assignees
    ADD CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: otps fk_user; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.otps
    ADD CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: param_logs fk_user; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.param_logs
    ADD CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: messages messages_media_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.messages
    ADD CONSTRAINT messages_media_id_fkey FOREIGN KEY (media_id) REFERENCES public.media(id) ON DELETE SET NULL;


--
-- Name: messages messages_sender_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.messages
    ADD CONSTRAINT messages_sender_id_fkey FOREIGN KEY (sender_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: sessions sessions_plan_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sessions
    ADD CONSTRAINT sessions_plan_id_fkey FOREIGN KEY (plan_id) REFERENCES public.plans(id) ON DELETE SET NULL;


--
-- Name: sessions sessions_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sessions
    ADD CONSTRAINT sessions_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: users users_avatar_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_avatar_id_fkey FOREIGN KEY (avatar_id) REFERENCES public.media(id);


--
-- Name: workout_logs workout_logs_exercise_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workout_logs
    ADD CONSTRAINT workout_logs_exercise_id_fkey FOREIGN KEY (exercise_id) REFERENCES public.exercises(id) ON DELETE CASCADE;


--
-- Name: workout_logs workout_logs_session_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workout_logs
    ADD CONSTRAINT workout_logs_session_id_fkey FOREIGN KEY (session_id) REFERENCES public.sessions(id) ON DELETE CASCADE;


--
-- Name: workout_logs_tags workout_logs_tags_tag_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workout_logs_tags
    ADD CONSTRAINT workout_logs_tags_tag_id_fkey FOREIGN KEY (tag_id) REFERENCES public.tags(id) ON DELETE CASCADE;


--
-- Name: workout_logs_tags workout_logs_tags_workout_log_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workout_logs_tags
    ADD CONSTRAINT workout_logs_tags_workout_log_id_fkey FOREIGN KEY (workout_log_id) REFERENCES public.workout_logs(id) ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--

