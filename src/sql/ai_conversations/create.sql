INSERT INTO ai_conversations (user_id, title)
VALUES ($1, $2)
RETURNING id, user_id, title, memory, summarized_until, created_at, updated_at
