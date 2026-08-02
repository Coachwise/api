UPDATE ai_messages SET actions = $2::jsonb, status = $3 WHERE id = $1
