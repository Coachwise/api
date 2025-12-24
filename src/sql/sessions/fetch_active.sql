SELECT id, COUNT(*) OVER () as total_count
FROM sessions
WHERE user_id = $1 AND status = 'ACTIVE'
ORDER BY started_at DESC
