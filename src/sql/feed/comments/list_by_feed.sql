SELECT *
FROM feed_comments
WHERE feed_id = $1
ORDER BY created_at ASC;
