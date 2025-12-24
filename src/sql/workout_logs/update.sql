UPDATE workout_logs
SET exercise_id = $2, exercise_name = $3, set_number = $4, reps = $5, weight = $6, rpe = $7, duration_seconds = $8, grade = $9, completed = $10, attempts = $11, notes = $12
WHERE id = $1
RETURNING *
