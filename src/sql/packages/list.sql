SELECT id, COUNT(*) OVER () AS total_count
FROM coach_packages
WHERE coach_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $2 OFFSET $3
