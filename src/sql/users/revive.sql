-- Undelete. Only ever called after an OTP is verified: proving you control the
-- number is what earns the account back.
UPDATE users SET deleted_at = NULL, updated_at = now()
WHERE id = $1
