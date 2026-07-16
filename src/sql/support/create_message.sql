INSERT INTO support_messages (ticket_id, sender, body)
VALUES ($1, $2, $3)
RETURNING *;
