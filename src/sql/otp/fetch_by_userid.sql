SELECT * from otps
WHERE user_id=$1
ORDER BY created_at DESC