SELECT cp.*,
       (SELECT COUNT(*) FROM coach_package_plans j WHERE j.package_id = cp.id) AS plan_count,
       COALESCE((
           SELECT jsonb_agg(jsonb_build_object('id', pl.id, 'name', pl.name) ORDER BY pl.name)
           FROM coach_package_plans j
           JOIN plans pl ON pl.id = j.plan_id
           WHERE j.package_id = cp.id
       ), '[]'::jsonb) AS plans
FROM coach_packages cp
WHERE cp.id IN (?) AND cp.deleted_at IS NULL
