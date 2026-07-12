-- Decide a pending application. The decision_token must match (the capability
-- link's secret); only PENDING applications can be decided.
UPDATE coach_applications
SET status = $3, review_note = $4, updated_at = now()
WHERE id = $1 AND decision_token = $2 AND status = 'PENDING'
RETURNING *;
