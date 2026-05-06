DELETE FROM plan_schedules WHERE id = $1 RETURNING id;
