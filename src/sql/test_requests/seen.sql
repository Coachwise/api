-- A coach acknowledges a submitted assessment. Works for requests they sent and
-- for self-assessments by athletes they're connected to (claims who reviewed it).
UPDATE test_requests tr
SET status = 'SEEN', coach_id = COALESCE(tr.coach_id, $2), seen_at = now(), updated_at = now()
WHERE tr.id = $1
  AND tr.status = 'SUBMITTED'
  AND (
    tr.coach_id = $2
    OR (tr.coach_id IS NULL AND EXISTS (
        SELECT 1 FROM connections cn
        WHERE cn.user1_id = LEAST($2::uuid, tr.athlete_id)
          AND cn.user2_id = GREATEST($2::uuid, tr.athlete_id)
    ))
  )
RETURNING *
