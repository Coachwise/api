CREATE EXTENSION IF NOT EXISTS "uuid-ossp" WITH SCHEMA public;
COMMENT ON EXTENSION "uuid-ossp" IS 'generate universally unique identifiers (UUIDs)';

CREATE TYPE public.sports AS ENUM (
    'FITNESS',
    'CLIMBING',
    'THERAPEUTIC'
);


CREATE TABLE media (
  id UUID NOT NULL DEFAULT public.uuid_generate_v4() PRIMARY KEY,
  user_id UUID,
  url TEXT NOT NULL UNIQUE,
  filename TEXT NOT NULL,
  created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE users (
  id UUID NOT NULL DEFAULT public.uuid_generate_v4() PRIMARY KEY,
  username VARCHAR(128) UNIQUE NOT NULL,
  password TEXT,
  first_name VARCHAR(128),
  last_name VARCHAR(128),
  email VARCHAR(128) UNIQUE NOT NULL,
  phone VARCHAR(128) UNIQUE,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE coaches (
    user_id uuid NOT NULL PRIMARY KEY,
    specialties public.sports[] NOT NULL,
    created_at timestamp without time zone DEFAULT now(),
    updated_at timestamp without time zone DEFAULT now(),
    CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);


CREATE TABLE exercises (
    id uuid DEFAULT public.uuid_generate_v4() PRIMARY KEY,
    user_id uuid,
    name VARCHAR(128) NOT NULL,
    description text NOT NULL,
    public boolean DEFAULT false,
    created_at timestamp without time zone DEFAULT now(),
    updated_at timestamp without time zone DEFAULT now(),
    CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
);

CREATE TABLE plans (
    id uuid DEFAULT public.uuid_generate_v4() PRIMARY KEY,
    user_id uuid NOT NULL,
    public boolean DEFAULT false,
    name character varying(128) NOT NULL,
    created_at timestamp without time zone DEFAULT now(),
    updated_at timestamp without time zone DEFAULT now(),
    CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);


CREATE TABLE plan_assignees (
    id uuid DEFAULT public.uuid_generate_v4() PRIMARY KEY,
    plan_id uuid NOT NULL,
    user_id uuid NOT NULL,
    due_at timestamp without time zone,
    created_at timestamp without time zone DEFAULT now(),
    updated_at timestamp without time zone DEFAULT now(),
    CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE plan_exercises (
    id uuid DEFAULT public.uuid_generate_v4() PRIMARY KEY,
    exercise_id uuid NOT NULL,
    plan_id uuid NOT NULL,
    exercise_order integer NOT NULL,
    rest_time bigint NOT NULL,
    created_at timestamp without time zone DEFAULT now(),
    CONSTRAINT fk_exercise FOREIGN KEY (exercise_id) REFERENCES exercises(id) ON DELETE CASCADE,
    CONSTRAINT fk_plan FOREIGN KEY (plan_id) REFERENCES plans(id) ON DELETE CASCADE
);


CREATE TABLE sets (
    id uuid DEFAULT public.uuid_generate_v4() PRIMARY KEY,
    name character varying(128),
    exercise_id uuid NOT NULL,
    set_number integer NOT NULL,
    rest_time bigint NOT NULL,
    rep_count integer,
    duration bigint,    
    created_at timestamp without time zone DEFAULT now(),
    updated_at timestamp without time zone DEFAULT now(),
    CONSTRAINT fk_exercise FOREIGN KEY (exercise_id) REFERENCES exercises(id) ON DELETE CASCADE,
    CONSTRAINT reps_check CHECK ((((rep_count IS NOT NULL) AND (duration IS NULL)) OR ((rep_count IS NULL) AND (duration IS NOT NULL))))
);