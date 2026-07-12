SELECT 1 FROM test_requests
WHERE test_id = $1 AND athlete_id = $2 AND coach_id IS NOT NULL
LIMIT 1
