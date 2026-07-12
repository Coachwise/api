INSERT INTO notifications (user_id, actor_id, type, entity_type, entity_id, data)
VALUES ($1, $2, $3, $4, $5, $6::jsonb)
RETURNING id
