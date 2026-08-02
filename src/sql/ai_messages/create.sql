INSERT INTO ai_messages (conversation_id, role, text, status)
VALUES ($1, $2, $3, $4)
RETURNING id
