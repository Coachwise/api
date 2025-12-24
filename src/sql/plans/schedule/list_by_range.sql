SELECT * FROM plan_schedule
WHERE user_id = $1
  AND scheduled_for >= $2
  AND scheduled_for <= $3
ORDER BY scheduled_for ASC, created_at ASC;
