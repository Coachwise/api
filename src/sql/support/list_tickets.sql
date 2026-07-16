-- The user's tickets, most recently active first, each with a preview of its last
-- message. count(*) OVER() carries the total for pagination in one round trip.
SELECT t.*,
       lm.body        AS last_body,
       lm.sender      AS last_sender,
       count(*) OVER() AS total_count
FROM support_tickets t
LEFT JOIN LATERAL (
    SELECT body, sender
    FROM support_messages m
    WHERE m.ticket_id = t.id
    ORDER BY m.created_at DESC
    LIMIT 1
) lm ON true
WHERE t.user_id = $1
ORDER BY t.last_message_at DESC
LIMIT $2 OFFSET $3;
