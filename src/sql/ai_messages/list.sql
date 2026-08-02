SELECT id, conversation_id, role, text, actions, status, model,
       prompt_tokens, completion_tokens, total_tokens, created_at
FROM ai_messages
WHERE conversation_id = $1
ORDER BY created_at ASC
