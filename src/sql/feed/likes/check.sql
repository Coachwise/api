SELECT TRUE AS liked
FROM feed_likes
WHERE feed_id = $1 AND user_id = $2
LIMIT 1;
