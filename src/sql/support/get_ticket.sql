-- Ownership-scoped: a user only ever sees their own ticket.
SELECT * FROM support_tickets WHERE id = $1 AND user_id = $2;
