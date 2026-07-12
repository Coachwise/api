SELECT id, requester_id, status, created_at,
       COUNT(*) OVER () AS total_count
FROM connection_requests
WHERE addressee_id = $1 AND status = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;
