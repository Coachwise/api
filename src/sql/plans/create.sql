INSERT INTO plans (user_id, public, name)
VALUES ($1, $2, $3)
RETURNING *
