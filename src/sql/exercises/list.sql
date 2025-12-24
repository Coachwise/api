SELECT id, COUNT(*) OVER () as total_count
FROM exercises
WHERE (COALESCE($1::boolean, public) = public)
  AND ($2 = '' OR LOWER(name) LIKE '%' || LOWER($2) || '%')
ORDER BY created_at DESC;
