DELETE FROM achievements WHERE id = $1 AND coach_id = $2 RETURNING id
