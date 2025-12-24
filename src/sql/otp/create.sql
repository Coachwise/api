WITH expired AS (
  UPDATE otps
  SET expired_at = NOW()
  WHERE user_id = $1 AND perpose = $3 AND is_verified = false
)
INSERT INTO otps(user_id, code, perpose, email)
VALUES ($1, $2, $3, $4)
RETURNING *
