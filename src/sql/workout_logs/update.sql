UPDATE workout_logs
SET reps = $2, weight = $3, rpe = $4, duration_seconds = $5, grade = $6, completed = $7, attempts = $8, notes = $9, distance = $10, height = $11
WHERE id = $1
RETURNING *
