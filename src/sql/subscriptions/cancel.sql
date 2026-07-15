-- Coach ends a client's subscription. The row is kept (soft-deleted) and records
-- who ended it and why: a refund is calculated from it, and a dispute is argued
-- from it.
UPDATE package_subscriptions
SET status = 'CANCELED',
    canceled_at = now(),
    canceled_by = $3,
    cancel_reason = $4,
    deleted_at = now(),
    updated_at = now()
WHERE package_id = $1 AND client_id = $2 AND deleted_at IS NULL
RETURNING *
