SELECT id, COUNT(*) OVER () as total_count
FROM plan_schedules
WHERE user_id = $1 AND status = 'ACTIVE' AND scheduled_for >= CURRENT_DATE
ORDER BY scheduled_for ASC
LIMIT $2 OFFSET $3
