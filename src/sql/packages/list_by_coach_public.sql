SELECT id
FROM coach_packages
WHERE coach_id = $1 AND is_active = true
ORDER BY popular DESC, created_at DESC
