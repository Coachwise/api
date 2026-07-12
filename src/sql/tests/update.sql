UPDATE tests SET name = $2, description = $3, public = $4, updated_at = now()
WHERE id = $1
RETURNING *
