INSERT INTO feed_tags (feed_id, tag_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;
