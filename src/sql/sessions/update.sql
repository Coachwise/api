UPDATE sessions
SET status = $2, ended_at = $3, notes = $4, intensity = $5, quality = $6, updated_at = NOW()
WHERE id = $1
RETURNING *
