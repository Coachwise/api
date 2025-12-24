INSERT INTO feed_likes (feed_id, user_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING
RETURNING *;
