SELECT id, user_id, title, memory, summarized_until, created_at, updated_at
FROM ai_conversations
WHERE id = $1 AND user_id = $2
