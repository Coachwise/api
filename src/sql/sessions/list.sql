SELECT id, COUNT(*) OVER () as total_count
FROM sessions
WHERE user_id = $1
ORDER BY started_at DESC
LIMIT $2 OFFSET $3
