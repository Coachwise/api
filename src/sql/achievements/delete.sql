UPDATE achievements SET deleted_at = now()
WHERE id = $1 AND coach_id = $2 AND deleted_at IS NULL
RETURNING id
