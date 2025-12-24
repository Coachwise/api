SELECT *
FROM feed_media
WHERE feed_id = $1
ORDER BY order_index ASC, created_at ASC;
