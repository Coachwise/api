SELECT id, COUNT(*) OVER () as total_count
FROM workout_logs
WHERE session_id = $1 AND deleted_at IS NULL
ORDER BY logged_at DESC
