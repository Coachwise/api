INSERT INTO coach_package_plans (package_id, plan_id)
VALUES ($1, $2)
ON CONFLICT (package_id, plan_id) DO NOTHING
