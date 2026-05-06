INSERT INTO plan_schedules (user_id, plan_id, scheduled_for, notes)
VALUES ($1, $2, $3, $4)
RETURNING *;
