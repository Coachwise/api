SELECT
    COALESCE(SUM(amount) FILTER (WHERE available_at <= now()), 0) AS available,
    COALESCE(SUM(amount) FILTER (WHERE amount > 0 AND available_at > now()), 0) AS pending
FROM wallet_transactions
WHERE wallet_id = $1;
