INSERT INTO workout_logs (session_id, exercise_id, exercise_name, set_number, reps, weight, rpe, duration_seconds, grade, completed, attempts, notes)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING *
