UPDATE connection_requests
SET status = 'ACCEPTED', updated_at = now()
WHERE id = $1 AND addressee_id = $2
RETURNING *;
