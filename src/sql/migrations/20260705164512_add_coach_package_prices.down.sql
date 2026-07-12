ALTER TABLE public.platform_settings DROP COLUMN IF EXISTS one_time_pro_months;
DROP TABLE IF EXISTS public.coach_package_prices;
ALTER TABLE public.coach_packages DROP CONSTRAINT IF EXISTS coach_packages_billing_type_chk;
ALTER TABLE public.coach_packages DROP COLUMN IF EXISTS billing_type;
