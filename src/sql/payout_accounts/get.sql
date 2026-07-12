SELECT * FROM payout_accounts
WHERE user_id = $1 AND currency = $2;
