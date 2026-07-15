SELECT client_id AS id, COUNT(*) OVER () AS total_count
FROM package_subscriptions
WHERE coach_id = $1 AND status = 'ACTIVE' AND deleted_at IS NULL
GROUP BY client_id
ORDER BY MAX(created_at) DESC
LIMIT $2 OFFSET $3
