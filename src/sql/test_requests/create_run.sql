-- One run of an athlete's own protocol (test): a self-submitted request.
INSERT INTO test_requests (test_id, athlete_id, status, submitted_at)
VALUES ($1, $2, 'SUBMITTED', now())
RETURNING *
