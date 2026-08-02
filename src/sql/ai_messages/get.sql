SELECT id, conversation_id, role, text, actions, status, model,
       prompt_tokens, completion_tokens, total_tokens, created_at
FROM ai_messages
WHERE id = $1 AND conversation_id = $2
