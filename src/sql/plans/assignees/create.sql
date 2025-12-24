INSERT INTO plan_assignees (plan_id, user_id, assigner)
VALUES ($1, $2, $3)
RETURNING *
