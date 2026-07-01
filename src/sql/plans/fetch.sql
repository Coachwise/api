SELECT p.*,
       (SELECT COUNT(*) FROM plan_exercises pe WHERE pe.plan_id = p.id) AS exercise_count,
       (SELECT (COALESCE(SUM(
           (CASE WHEN s.duration IS NOT NULL THEN s.duration ELSE COALESCE(s.rep_count, 0) * 6000000000 END)
           + s.rest_time
       ), 0))::bigint / 1000000000
        FROM plan_exercises pe JOIN sets s ON s.exercise_id = pe.exercise_id
        WHERE pe.plan_id = p.id) AS estimated_seconds,
       row_to_json(owner.*) AS "user"
FROM plans p
LEFT JOIN users owner ON owner.id = p.user_id
WHERE p.id IN (?)
ORDER BY p.created_at DESC
