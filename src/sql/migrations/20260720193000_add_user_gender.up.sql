SET search_path TO public;

-- Gender is collected at onboarding (drives the default UI theme) and editable
-- in profile settings. UNSPECIFIED keeps existing rows valid and lets a user
-- decline to answer.
CREATE TYPE genders AS ENUM ('MALE', 'FEMALE', 'UNSPECIFIED');

ALTER TABLE users
ADD COLUMN gender genders DEFAULT 'UNSPECIFIED'::genders NOT NULL;
