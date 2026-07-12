INSERT INTO payments
    (order_id, user_id, wallet_id, amount, currency, provider, provider_ref, status)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;
