INSERT INTO coach_package_prices (package_id, currency, amount)
VALUES ($1, $2, $3)
ON CONFLICT (package_id, currency) DO UPDATE SET amount = EXCLUDED.amount, updated_at = now();
