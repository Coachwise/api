INSERT INTO sessions (user_id, session_type, plan_id, status, started_at, notes)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *
