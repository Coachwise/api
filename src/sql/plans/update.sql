UPDATE plans
SET name=$2,
    public=$3,
    updated_at=NOW()
WHERE id=$1
RETURNING *
