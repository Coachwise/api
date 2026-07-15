SELECT p.*
FROM plans p
WHERE p.id = $1 AND p.deleted_at IS NULL AND (
    p.user_id = $2
    OR p.public = true
    OR EXISTS (SELECT 1 FROM plan_assignees pa WHERE pa.plan_id = p.id AND pa.user_id = $2)
)
