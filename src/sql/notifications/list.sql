SELECT id, COUNT(*) OVER () AS total_count
FROM notifications
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3
