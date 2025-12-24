INSERT INTO chats (type, name, owner_id)
VALUES ('CHANNEL', $2, $1)
RETURNING *;
