-- 1 if exercise $1 appears in any plan assigned to user $2 (used to let a client
-- open a coach's personal exercise that is part of a plan assigned to them).
SELECT 1
FROM plan_exercises pe
JOIN plan_assignees pa ON pa.plan_id = pe.plan_id
WHERE pe.exercise_id = $1
  AND pa.user_id = $2
LIMIT 1;
