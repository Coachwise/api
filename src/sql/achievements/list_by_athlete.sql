SELECT * FROM achievements WHERE athlete_id = $1 AND deleted_at IS NULL ORDER BY created_at DESC
