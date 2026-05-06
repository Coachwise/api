DELETE FROM workout_logs wl
USING sessions s
WHERE wl.id = $1
AND wl.session_id = s.id
AND s.user_id = $2
RETURNING wl.id;
