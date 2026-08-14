INSERT INTO plan_exercises (exercise_id, plan_id, exercise_order, rest_time, intensity,
    rounds, round_rest, round_duration)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *
