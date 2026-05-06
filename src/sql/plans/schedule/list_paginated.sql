SELECT id, COUNT(*) OVER () as total_count
FROM plan_schedules
WHERE user_id = $1
ORDER BY scheduled_for ASC, created_at ASC
LIMIT $2 OFFSET $3
