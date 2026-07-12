INSERT INTO payout_accounts
    (user_id, currency, method, account_holder, card_number, iban, bank_name, swift, stripe_account_id, status, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, now())
ON CONFLICT (user_id, currency) DO UPDATE SET
    method            = EXCLUDED.method,
    account_holder    = EXCLUDED.account_holder,
    card_number       = EXCLUDED.card_number,
    iban              = EXCLUDED.iban,
    bank_name         = EXCLUDED.bank_name,
    swift             = EXCLUDED.swift,
    stripe_account_id = EXCLUDED.stripe_account_id,
    status            = EXCLUDED.status,
    updated_at        = now()
RETURNING *;
