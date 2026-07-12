INSERT INTO tests (coach_id, name, description, public)
VALUES ($1, $2, $3, $4)
RETURNING *
