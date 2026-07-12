-- A coach sees their own requests plus self-assessments by connected athletes.
SELECT tr.id, COUNT(*) OVER () AS total_count
FROM test_requests tr
WHERE ($2 = '' OR tr.status = $2)
  AND (
    tr.coach_id = $1
    OR (tr.coach_id IS NULL AND EXISTS (
        SELECT 1 FROM connections cn
        WHERE cn.user1_id = LEAST($1::uuid, tr.athlete_id)
          AND cn.user2_id = GREATEST($1::uuid, tr.athlete_id)
    ))
  )
ORDER BY tr.created_at DESC
LIMIT $3 OFFSET $4
