SELECT
    f.id,
    COUNT(*) OVER () AS total_count,
    COALESCE(likes.count, 0) AS like_count,
    COALESCE(comments.count, 0) AS comment_count,
    EXISTS(
        SELECT 1 FROM feed_likes fl2 WHERE fl2.feed_id = f.id AND fl2.user_id = $1
    ) AS liked
FROM feeds f
LEFT JOIN LATERAL (
    SELECT COUNT(*) FROM feed_likes fl WHERE fl.feed_id = f.id
) AS likes(count) ON TRUE
LEFT JOIN LATERAL (
    SELECT COUNT(*) FROM feed_comments fc WHERE fc.feed_id = f.id
) AS comments(count) ON TRUE
WHERE f.deleted_at IS NULL
    AND (f.visibility = 'PUBLIC' OR f.user_id = $1)
ORDER BY f.created_at DESC
LIMIT $2 OFFSET $3;
