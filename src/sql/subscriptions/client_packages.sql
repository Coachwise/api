SELECT ps.client_id, cp.id AS package_id, cp.name AS package_name, ps.created_at
FROM package_subscriptions ps
JOIN coach_packages cp ON cp.id = ps.package_id
WHERE ps.coach_id = $1 AND ps.status = 'ACTIVE' AND ps.deleted_at IS NULL AND cp.deleted_at IS NULL
ORDER BY ps.created_at DESC
