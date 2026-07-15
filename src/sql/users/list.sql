SELECT u.id, COUNT(*) OVER () AS total_count
FROM users u
WHERE ($1 = '' OR u.search_vector @@ to_tsquery('simple', $1))
  AND (NOT $2::boolean OR u.is_coach = true)
  AND u.id <> $3::uuid
  AND u.deleted_at IS NULL
ORDER BY
  CASE WHEN $1 <> '' THEN ts_rank(u.search_vector, to_tsquery('simple', $1)) END DESC NULLS LAST,
  u.username
LIMIT $4 OFFSET $5;
