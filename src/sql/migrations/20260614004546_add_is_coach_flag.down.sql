DROP TRIGGER IF EXISTS coaches_sync_is_coach ON public.coaches;
DROP FUNCTION IF EXISTS public.sync_user_is_coach();
DROP INDEX IF EXISTS public.idx_users_is_coach;
ALTER TABLE public.users DROP COLUMN IF EXISTS is_coach;
