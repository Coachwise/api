INSERT INTO wallets (owner_id, currency) VALUES (NULL, $1) RETURNING *;
