SELECT *
FROM messages
WHERE chat_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;
