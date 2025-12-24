UPDATE messages
SET read_at = NOW()
WHERE chat_id = $1
  AND sender_id <> $2
  AND read_at IS NULL;
