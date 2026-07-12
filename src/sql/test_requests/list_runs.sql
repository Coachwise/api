-- Actual runs (records submitted), excluding assignment markers (PENDING).
SELECT id, COUNT(*) OVER () AS total_count
FROM test_requests
WHERE test_id = $1 AND athlete_id = $2 AND status <> 'PENDING'
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;
