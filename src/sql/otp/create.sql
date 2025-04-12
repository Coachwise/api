INSERT INTO otps(user_id, code, perpose)
VALUES ($1, $2, $3)
RETURNING *