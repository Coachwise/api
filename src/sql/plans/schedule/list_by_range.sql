SELECT * FROM plan_schedules
WHERE user_id = $1
  AND status = 'ACTIVE'
  AND scheduled_for >= $2
  AND scheduled_for <= $3
ORDER BY scheduled_for ASC, created_at ASC;
