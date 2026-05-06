UPDATE plan_schedules
SET status = COALESCE($2, status),
    notes = COALESCE($3, notes),
    updated_at = NOW()
WHERE id = $1
RETURNING *;
