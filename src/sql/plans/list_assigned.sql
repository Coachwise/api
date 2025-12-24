SELECT p.*
FROM plans p
INNER JOIN plan_assignees pa ON pa.plan_id = p.id
WHERE pa.user_id = $1
