ALTER TABLE public.coach_package_prices DROP CONSTRAINT IF EXISTS coach_package_prices_currency_fk;
ALTER TABLE public.pro_prices DROP CONSTRAINT IF EXISTS pro_prices_currency_fk;
DROP TABLE IF EXISTS public.currencies;
