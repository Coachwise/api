UPDATE users
SET pro_until = GREATEST(COALESCE(pro_until, now()), now()) + make_interval(months => $2),
    updated_at = now()
WHERE id = $1
RETURNING pro_until;
