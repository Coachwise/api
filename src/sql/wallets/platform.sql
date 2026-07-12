SELECT * FROM wallets WHERE owner_id IS NULL AND currency = $1;
