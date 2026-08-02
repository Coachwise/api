SELECT id, conversation_id, role, text, actions, status, model,
       prompt_tokens, completion_tokens, total_tokens, created_at
FROM ai_messages
WHERE conversation_id = $1
  AND ($2::timestamp IS NULL OR created_at > $2)
ORDER BY created_at ASC
