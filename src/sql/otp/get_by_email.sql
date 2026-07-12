SELECT * FROM otps WHERE email = $1 ORDER BY created_at DESC LIMIT 1
