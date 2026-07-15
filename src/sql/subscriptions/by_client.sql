SELECT *, COUNT(*) OVER () AS total_count
FROM package_subscriptions
WHERE client_id = $1 AND status = 'ACTIVE' AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;
