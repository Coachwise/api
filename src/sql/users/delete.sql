-- The row stays: an OTP login later just clears deleted_at and the account is
-- back, keeping its email, phone and username.
UPDATE users SET deleted_at = now()
WHERE id = $1 AND deleted_at IS NULL
