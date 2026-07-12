INSERT INTO wallet_transactions
    (wallet_id, currency, amount, type, available_at, ref_type, ref_id, description)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;
