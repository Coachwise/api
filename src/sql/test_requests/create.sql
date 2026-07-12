INSERT INTO test_requests (test_id, coach_id, athlete_id, note)
VALUES ($1, $2, $3, $4)
RETURNING *
