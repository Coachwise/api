INSERT INTO chat_members (chat_id, user_id, role)
VALUES ($1, $2, $3)
ON CONFLICT (chat_id, user_id) DO UPDATE SET role = EXCLUDED.role
RETURNING *;
