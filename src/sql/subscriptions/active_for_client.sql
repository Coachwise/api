SELECT * FROM package_subscriptions
WHERE coach_id = $1 AND client_id = $2 AND status = 'ACTIVE'
LIMIT 1
