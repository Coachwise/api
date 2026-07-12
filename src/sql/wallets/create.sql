INSERT INTO wallets (owner_id, currency) VALUES ($1, $2) RETURNING *;
