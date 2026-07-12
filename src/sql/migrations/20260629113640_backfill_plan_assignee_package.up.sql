-- Backfill package origin for assignments created before plan_assignees.package_id
-- existed: if a client has an active subscription to a package that bundles the
-- assigned plan (and the coach is the assigner), stamp that package_id.
UPDATE public.plan_assignees pa
SET package_id = ps.package_id
FROM public.package_subscriptions ps
JOIN public.coach_package_plans cpp ON cpp.package_id = ps.package_id
WHERE pa.package_id IS NULL
  AND ps.status = 'ACTIVE'
  AND ps.client_id = pa.user_id
  AND ps.coach_id = pa.assigner
  AND cpp.plan_id = pa.plan_id;
