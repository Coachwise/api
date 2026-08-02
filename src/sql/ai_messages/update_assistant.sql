UPDATE ai_messages
SET text = $2, actions = $3::jsonb, status = $4, model = $5,
    prompt_tokens = $6, completion_tokens = $7, total_tokens = $8
WHERE id = $1
