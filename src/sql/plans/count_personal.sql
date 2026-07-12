SELECT COUNT(*) AS count FROM plans WHERE user_id = $1 AND public = false
