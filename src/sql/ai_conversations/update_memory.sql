UPDATE ai_conversations
SET memory = $2, summarized_until = $3, updated_at = now()
WHERE id = $1
