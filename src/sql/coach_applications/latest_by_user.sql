SELECT * FROM coach_applications
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT 1;
