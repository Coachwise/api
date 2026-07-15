-- Adds to the running refunded total. The order becomes REFUNDED once the whole
-- of it has been given back, PARTIALLY_REFUNDED while some of the term was used.
UPDATE orders
SET refunded_amount = refunded_amount + $2,
    status = CASE WHEN refunded_amount + $2 >= total THEN 'REFUNDED' ELSE 'PARTIALLY_REFUNDED' END,
    updated_at = now()
WHERE id = $1
RETURNING *
