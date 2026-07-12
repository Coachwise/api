-- Paginated IDs of plans visible to a user (own, public, assigned), optionally
-- restricted to public ($2) and filtered by a prefix tsquery ($3). Hydrated via
-- plans/fetch on the ids.
SELECT p.id, COUNT(*) OVER () AS total_count
FROM plans p
WHERE (
        p.public = true
        OR (NOT $2::boolean AND (
             p.user_id = $1
             OR EXISTS (SELECT 1 FROM plan_assignees pa WHERE pa.plan_id = p.id AND pa.user_id = $1)
        ))
      )
  AND ($3 = '' OR p.search_vector @@ to_tsquery('simple', $3))
ORDER BY
  CASE WHEN $3 <> '' THEN ts_rank(p.search_vector, to_tsquery('simple', $3)) END DESC NULLS LAST,
  p.created_at DESC
LIMIT $4 OFFSET $5;
