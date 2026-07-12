INSERT INTO payouts (coach_id, wallet_id, amount, currency, status, note)
VALUES ($1, $2, $3, $4, 'REQUESTED', $5)
RETURNING *;
