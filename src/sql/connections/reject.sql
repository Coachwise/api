UPDATE connection_requests
SET status = 'REJECTED', updated_at = now()
WHERE id = $1 AND addressee_id = $2
RETURNING *;
