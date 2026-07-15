SELECT * FROM users WHERE phone = $1 AND deleted_at IS NULL LIMIT 1
