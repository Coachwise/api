
CREATE TYPE public.units AS ENUM (
  'KG',
  'CM',
  'SECOND',
  'COUNT'
);

CREATE TYPE public.sides AS ENUM (
  'LEFT',
  'RIGHT',
  'GENERAL'
);

CREATE TABLE params (
  id UUID NOT NULL DEFAULT public.uuid_generate_v4() PRIMARY KEY,
  name TEXT UNIQUE NOT NULL,
  description TEXT,
  unit units NOT NULL DEFAULT 'COUNT',
  side sides NOT NULL DEFAULT 'GENERAL',
  available_sports public.sports[] NOT NULL,
  updated_at TIMESTAMP DEFAULT NOW(),
  created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE param_logs (
  id UUID NOT NULL DEFAULT public.uuid_generate_v4() PRIMARY KEY,
  value NUMERIC(10, 2) NOT NULL,
  note TEXT,
  user_id uuid NOT NULL,
  param_id uuid NOT NULL,
  updated_at TIMESTAMP DEFAULT NOW(),
  created_at TIMESTAMP DEFAULT NOW(),
  CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  CONSTRAINT fk_param FOREIGN KEY (param_id) REFERENCES params(id) ON DELETE CASCADE
);