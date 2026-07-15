SELECT COUNT(DISTINCT client_id) FROM package_subscriptions
WHERE coach_id = $1 AND status = 'ACTIVE' AND deleted_at IS NULL
