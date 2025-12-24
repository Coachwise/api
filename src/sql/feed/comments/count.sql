SELECT COUNT(*) AS count
FROM feed_comments
WHERE feed_id = $1;
