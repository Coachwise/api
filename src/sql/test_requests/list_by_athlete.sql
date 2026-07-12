SELECT id, COUNT(*) OVER () AS total_count
FROM test_requests
WHERE athlete_id = $1 AND ($2 = '' OR status = $2)
ORDER BY created_at DESC
LIMIT $3 OFFSET $4
