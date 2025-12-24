INSERT INTO plan_exercises (exercise_id, plan_id, exercise_order, rest_time)
VALUES ($1, $2, $3, $4)
RETURNING *
