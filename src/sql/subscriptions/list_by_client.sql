SELECT id FROM package_subscriptions
WHERE client_id = $1 AND status = 'ACTIVE' AND deleted_at IS NULL
ORDER BY created_at DESC
