CREATE TYPE user_status AS ENUM ('ACTIVE', 'INACTIVE', 'SUSPENDED');
CREATE TYPE otp_perposes AS ENUM ('AUTH', 'FORGET_PASSWORD');

CREATE TABLE otps (
  id UUID NOT NULL DEFAULT public.uuid_generate_v4() PRIMARY KEY,
  user_id UUID NOT NULL,
  code integer NOT NULL,
  created_at TIMESTAMP DEFAULT NOW(),
  expired_at timestamp with time zone DEFAULT (now() + '00:10:00'::interval) NOT NULL,
  CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

--Alters
ALTER TABLE users
ADD COLUMN status user_status DEFAULT 'INACTIVE' NOT NULL,
ADD COLUMN password_expired boolean DEFAULT false NOT NULL;


ALTER TABLE otps
ADD COLUMN is_verified boolean DEFAULT false NOT NULL,
ADD COLUMN perpose otp_perposes DEFAULT 'AUTH' NOT NULL;

--Tables
CREATE TABLE tokens_blacklist (
  id UUID NOT NULL DEFAULT public.uuid_generate_v4() PRIMARY KEY,
  token TEXT UNIQUE,
  expired_at TIMESTAMP DEFAULT NOW()
);
