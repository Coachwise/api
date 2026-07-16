-- The turn guard for a user reply, atomic. Flips the turn to the admin only when
-- the ticket is open, owned by this user, and it is genuinely the user's turn.
-- Zero rows back means the send is not allowed; the caller decides why (not
-- found / closed / not your turn) from a prior fetch.
UPDATE support_tickets
SET turn = 'ADMIN', last_message_at = now(), updated_at = now()
WHERE id = $1 AND user_id = $2 AND status = 'OPEN' AND turn = 'USER'
RETURNING id;
