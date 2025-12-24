DELETE FROM plan_schedule WHERE id = $1 RETURNING id;
