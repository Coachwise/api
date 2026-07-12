-- Distinct protocols a coach has assigned to this athlete (assignment = a coach
-- created test_request), most recently assigned first.
SELECT tr.test_id AS id, COUNT(*) OVER () AS total_count
FROM test_requests tr
WHERE tr.athlete_id = $1 AND tr.coach_id IS NOT NULL AND tr.test_id IS NOT NULL
GROUP BY tr.test_id
ORDER BY MAX(tr.created_at) DESC
LIMIT $2 OFFSET $3
