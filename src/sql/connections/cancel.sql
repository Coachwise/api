DELETE FROM connection_requests
WHERE requester_id = $1 AND addressee_id = $2
RETURNING id;
