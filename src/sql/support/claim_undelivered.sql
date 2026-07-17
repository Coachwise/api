-- The worker's atomic claim. Admin replies are written straight to the DB by the
-- admin panel, which can't reach the event bus, so the worker finds the ones it
-- hasn't pushed yet and stamps delivered_at in the same statement. RETURNING the
-- claimed rows (joined to the ticket for the recipient) means two workers never
-- deliver the same message — each row is returned to exactly one claimer.
UPDATE support_messages m
SET delivered_at = now()
FROM support_tickets t
WHERE m.ticket_id = t.id
  AND m.delivered_at IS NULL
  AND m.sender <> 'USER'
RETURNING m.id, m.ticket_id, t.user_id, m.sender, m.body;
