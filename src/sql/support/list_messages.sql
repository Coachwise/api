SELECT * FROM support_messages
WHERE ticket_id = $1
ORDER BY created_at ASC;
