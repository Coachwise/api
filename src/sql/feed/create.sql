INSERT INTO feeds (user_id, body, location, visibility)
VALUES ($1, $2, $3, $4)
RETURNING *;
