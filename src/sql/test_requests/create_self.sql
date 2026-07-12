-- An athlete's self-assessment: no coach, no template, already done.
INSERT INTO test_requests (athlete_id, name, status, submitted_at)
VALUES ($1, $2, 'SUBMITTED', now())
RETURNING *
