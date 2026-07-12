ALTER TABLE public.plan_assignees DROP CONSTRAINT IF EXISTS plan_assignees_package_fk;
ALTER TABLE public.plan_assignees DROP COLUMN IF EXISTS package_id;
