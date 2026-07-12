ALTER TABLE public.users
    DROP COLUMN IF EXISTS website,
    DROP COLUMN IF EXISTS instagram,
    DROP COLUMN IF EXISTS birthday;
