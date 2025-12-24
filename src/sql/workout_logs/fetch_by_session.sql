SELECT id, COUNT(*) OVER () as total_count
FROM workout_logs
WHERE session_id = $1
ORDER BY logged_at DESC
