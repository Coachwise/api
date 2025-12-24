INSERT INTO messages (chat_id, sender_id, body, media_id)
VALUES ($1, $2, $3, $4)
RETURNING *;
