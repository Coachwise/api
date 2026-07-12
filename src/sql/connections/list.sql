SELECT CASE WHEN user1_id = $1 THEN user2_id ELSE user1_id END AS id,
       COUNT(*) OVER () AS total_count
FROM connections
WHERE user1_id = $1 OR user2_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;
