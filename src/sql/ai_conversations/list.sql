SELECT id, user_id, title, memory, summarized_until, created_at, updated_at,
       COUNT(*) OVER () AS total_count
FROM ai_conversations
WHERE user_id = $1
ORDER BY updated_at DESC
LIMIT $2 OFFSET $3
