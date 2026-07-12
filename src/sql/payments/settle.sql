UPDATE payments
SET status       = $2,
    provider_ref = COALESCE(NULLIF($3, ''), provider_ref),
    updated_at   = now()
WHERE id = $1 AND status = 'PENDING'
RETURNING *;
