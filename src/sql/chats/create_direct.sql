INSERT INTO chats (type, owner_id, chat_key)
VALUES ('DIRECT', $1, $2)
RETURNING *;
