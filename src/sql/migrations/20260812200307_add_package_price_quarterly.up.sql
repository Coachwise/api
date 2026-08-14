-- A 3-month price point, alongside monthly/annual/one-time. Nullable: an unset
-- price means the coach doesn't offer that duration.
ALTER TABLE public.coach_packages ADD COLUMN IF NOT EXISTS price_quarterly integer;
