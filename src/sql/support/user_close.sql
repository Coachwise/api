-- A user closes their own open ticket. Zero rows back = not theirs, or already
-- closed. The status flip and the closing marker are one transaction (below).
UPDATE support_tickets
SET status = 'CLOSED', updated_at = now(), last_message_at = now()
WHERE id = $1 AND user_id = $2 AND status = 'OPEN'
RETURNING *;
