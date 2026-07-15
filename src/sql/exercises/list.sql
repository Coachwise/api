SELECT id, COUNT(*) OVER () AS total_count
FROM exercises
WHERE deleted_at IS NULL
  AND (COALESCE($1::boolean, public) = public)
  AND ($2 = '' OR search_vector @@ to_tsquery('simple', $2))
  AND ($5 = '' OR category_id = (SELECT id FROM exercise_categories WHERE slug = $5))
  AND ($6 = '' OR sport_type::text = $6)
ORDER BY
  CASE WHEN $2 <> '' THEN ts_rank(search_vector, to_tsquery('simple', $2)) END DESC NULLS LAST,
  created_at DESC
LIMIT $3 OFFSET $4;
