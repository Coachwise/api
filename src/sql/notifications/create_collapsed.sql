-- Chat notifications collapse: bump the sender's existing unread row instead of
-- adding one per message, and only insert when there is none to bump.
WITH bumped AS (
    UPDATE notifications
    SET created_at = now(), entity_type = $4, entity_id = $5, data = $6::jsonb
    WHERE user_id = $1 AND actor_id = $2 AND type = $3 AND read = false
    RETURNING id
)
INSERT INTO notifications (user_id, actor_id, type, entity_type, entity_id, data)
SELECT $1, $2, $3, $4, $5, $6::jsonb
WHERE NOT EXISTS (SELECT 1 FROM bumped)
RETURNING id;
