-- Extra profile fields shown on a user's profile. All nullable.
ALTER TABLE public.users
    ADD COLUMN IF NOT EXISTS website   text,
    ADD COLUMN IF NOT EXISTS instagram text,
    ADD COLUMN IF NOT EXISTS birthday  date;
