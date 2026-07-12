INSERT INTO connection_requests (requester_id, addressee_id, status, updated_at)
VALUES ($1, $2, 'PENDING', now())
ON CONFLICT (requester_id, addressee_id)
DO UPDATE SET status = 'PENDING', updated_at = now()
RETURNING *;
