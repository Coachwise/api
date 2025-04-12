UPDATE otps
SET is_verified=true
WHERE user_id=$1 AND code=$2 AND is_verified=false
RETURNING *