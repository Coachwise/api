-- Denormalized is_coach flag on users, kept in sync with the coaches table.
-- Lets every user query (incl. /users/me and search) expose coach status and
-- makes the coach-only search filter indexable without a join.

ALTER TABLE public.users
    ADD COLUMN IF NOT EXISTS is_coach boolean NOT NULL DEFAULT false;

-- Backfill from existing coaches.
UPDATE public.users u
SET is_coach = true
WHERE EXISTS (SELECT 1 FROM public.coaches c WHERE c.user_id = u.id);

CREATE INDEX IF NOT EXISTS idx_users_is_coach ON public.users (is_coach);

-- Keep users.is_coach in sync when a coaches row is added or removed.
CREATE OR REPLACE FUNCTION public.sync_user_is_coach() RETURNS trigger AS $$
BEGIN
    IF (TG_OP = 'INSERT') THEN
        UPDATE public.users SET is_coach = true WHERE id = NEW.user_id;
    ELSIF (TG_OP = 'DELETE') THEN
        UPDATE public.users SET is_coach = false WHERE id = OLD.user_id;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS coaches_sync_is_coach ON public.coaches;
CREATE TRIGGER coaches_sync_is_coach
    AFTER INSERT OR DELETE ON public.coaches
    FOR EACH ROW EXECUTE FUNCTION public.sync_user_is_coach();
