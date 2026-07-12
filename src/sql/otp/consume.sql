-- Atomically validate + consume the latest matching OTP: verifies code, checks
-- not-expired and not-already-used, and marks it used. RETURNING = valid.
UPDATE otps SET is_verified = true
WHERE id = (
    SELECT id FROM otps
    WHERE user_id = $1 AND perpose = $2 AND code = $3
      AND is_verified = false AND expired_at > now()
    ORDER BY created_at DESC
    LIMIT 1
)
RETURNING id
