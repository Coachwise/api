INSERT INTO plan_schedule (user_id, plan_id, scheduled_for, status, notes)
VALUES ($1, $2, $3, COALESCE($4, 'PENDING'), $5)
RETURNING *;
