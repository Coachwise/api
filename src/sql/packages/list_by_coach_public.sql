SELECT id
FROM coach_packages
WHERE coach_id = $1 AND is_active = true AND deleted_at IS NULL
ORDER BY popular DESC, created_at DESC
