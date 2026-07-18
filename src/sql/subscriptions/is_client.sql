SELECT 1
FROM package_subscriptions
WHERE coach_id = $1 AND client_id = $2 AND status = 'ACTIVE' AND deleted_at IS NULL
LIMIT 1
