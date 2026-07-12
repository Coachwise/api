UPDATE test_requests SET status = 'SUBMITTED', submitted_at = now(), updated_at = now()
WHERE id = $1 AND athlete_id = $2 AND status = 'PENDING'
RETURNING *
