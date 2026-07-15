SELECT pa.user_id, pl.id AS plan_id, pl.name AS plan_name, pa.package_id, pa.created_at
FROM plan_assignees pa
JOIN plans pl ON pl.id = pa.plan_id AND pl.deleted_at IS NULL
WHERE pa.assigner = $1
ORDER BY pa.created_at DESC
