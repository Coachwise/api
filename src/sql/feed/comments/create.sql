INSERT INTO feed_comments (feed_id, user_id, body)
VALUES ($1, $2, $3)
RETURNING *;
