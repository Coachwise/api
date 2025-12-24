UPDATE plan_schedule
SET status = COALESCE($2, status),
    notes = COALESCE($3, notes),
    updated_at = NOW()
WHERE id = $1
RETURNING *;
