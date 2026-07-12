-- Conversation list for a user: one row per DIRECT chat they belong to that has
-- at least one message, with the peer, the last message, and the unread count.
SELECT
    c.id                       AS chat_id,
    cm2.user_id                AS peer_id,
    lm.body                    AS last_body,
    lm.created_at              AS last_at,
    lm.sender_id               AS last_sender_id,
    COALESCE(uc.unread, 0)     AS unread_count,
    COUNT(*) OVER ()           AS total_count
FROM chat_members cm
JOIN chats c ON c.id = cm.chat_id AND c.type = 'DIRECT'
JOIN chat_members cm2 ON cm2.chat_id = c.id AND cm2.user_id <> $1
JOIN LATERAL (
    SELECT body, created_at, sender_id
    FROM messages m
    WHERE m.chat_id = c.id
    ORDER BY m.created_at DESC
    LIMIT 1
) lm ON TRUE
LEFT JOIN LATERAL (
    SELECT COUNT(*) AS unread
    FROM messages m
    WHERE m.chat_id = c.id AND m.sender_id <> $1 AND m.read_at IS NULL
) uc ON TRUE
WHERE cm.user_id = $1
ORDER BY lm.created_at DESC
LIMIT $2 OFFSET $3;
