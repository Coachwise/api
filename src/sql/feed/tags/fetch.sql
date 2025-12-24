SELECT
    t.id,
    COUNT(*) OVER () AS total_count
FROM feed_tags ft
JOIN tags t ON ft.tag_id = t.id
WHERE ft.feed_id = $1;
