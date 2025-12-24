SELECT COUNT(*) AS count
FROM feed_likes
WHERE feed_id = $1;
