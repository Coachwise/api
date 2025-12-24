INSERT INTO feed_media (feed_id, kind, url, thumbnail_url, order_index)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;
