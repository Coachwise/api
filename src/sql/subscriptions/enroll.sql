INSERT INTO package_subscriptions (package_id, coach_id, client_id, status)
VALUES ($1, $2, $3, 'ACTIVE')
ON CONFLICT (package_id, client_id)
DO UPDATE SET status = 'ACTIVE', updated_at = now()
RETURNING *
