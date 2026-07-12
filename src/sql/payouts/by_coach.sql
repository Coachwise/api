SELECT *, COUNT(*) OVER () AS total_count
FROM payouts
WHERE coach_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;
