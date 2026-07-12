SELECT id FROM package_subscriptions
WHERE client_id = $1 AND status = 'ACTIVE'
ORDER BY created_at DESC
