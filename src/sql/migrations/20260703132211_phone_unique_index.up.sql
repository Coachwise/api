-- Phone is the login handle for phone+OTP auth, so it must be unique — but only
-- among rows that have one (existing email accounts keep NULL phone).
CREATE UNIQUE INDEX IF NOT EXISTS users_phone_unique ON public.users (phone) WHERE phone IS NOT NULL;
