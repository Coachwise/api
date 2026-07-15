-- Re-subscribing a previously cancelled client revives their row rather than
-- inserting a second one, so (package_id, client_id) stays unique.
INSERT INTO package_subscriptions (package_id, coach_id, client_id, status, ends_at, order_id)
VALUES ($1, $2, $3, 'ACTIVE', $4, $5)
ON CONFLICT (package_id, client_id)
DO UPDATE SET status = 'ACTIVE',
              ends_at = EXCLUDED.ends_at,
              order_id = EXCLUDED.order_id,
              deleted_at = NULL,
              canceled_at = NULL,
              canceled_by = NULL,
              cancel_reason = NULL,
              updated_at = now()
RETURNING *
