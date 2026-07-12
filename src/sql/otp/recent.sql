-- Is there a code issued for this user+purpose within the cooldown window?
SELECT 1 FROM otps
WHERE user_id = $1 AND perpose = $2 AND created_at > now() - make_interval(secs => $3::int)
LIMIT 1
