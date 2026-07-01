INSERT INTO plan_assignees (plan_id, user_id, assigner, package_id)
VALUES ($1, $2, $3, $4)
ON CONFLICT (plan_id, user_id)
DO UPDATE SET assigner = EXCLUDED.assigner, package_id = EXCLUDED.package_id
RETURNING *
