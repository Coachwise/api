INSERT INTO media (user_id, url, filename, size_bytes)
VALUES ($1, $2, $3, $4)
RETURNING *;
