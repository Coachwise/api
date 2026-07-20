SET search_path TO public;

ALTER TABLE users DROP COLUMN IF EXISTS gender;
DROP TYPE IF EXISTS genders;
