UPDATE workout_logs wl SET deleted_at = now()
FROM sessions s
WHERE wl.id = $1
AND wl.session_id = s.id
AND s.user_id = $2
AND wl.deleted_at IS NULL
RETURNING wl.id;
