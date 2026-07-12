-- Coach income = SALE credits to the wallet. `total` is all-time cumulative;
-- `month` is the CURRENT CALENDAR month (date_trunc, not a rolling 30 days),
-- by created_at — i.e. when earned, independent of escrow/available_at.
SELECT
  COALESCE(SUM(amount), 0) AS total,
  COALESCE(SUM(amount) FILTER (WHERE created_at >= date_trunc('month', now())), 0) AS month
FROM wallet_transactions
WHERE wallet_id = $1 AND type = 'SALE';
