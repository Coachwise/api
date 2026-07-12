-- Best set per exercise from submitted assessments (coach-requested or self).
-- DISTINCT ON keeps reps + weight paired (e.g. 50kg x 3) instead of maxing each
-- column independently. Records count as soon as they're submitted (no approval).
SELECT DISTINCT ON (wl.exercise_id)
       wl.exercise_id,
       e.name AS exercise_name,
       wl.weight AS best_weight,
       wl.reps AS best_reps,
       wl.duration_seconds AS best_time
FROM workout_logs wl
JOIN test_requests tr ON tr.id = wl.test_request_id
JOIN exercises e ON e.id = wl.exercise_id
WHERE tr.athlete_id = $1 AND tr.status IN ('SUBMITTED', 'SEEN')
ORDER BY wl.exercise_id,
         wl.weight DESC NULLS LAST,
         wl.reps DESC NULLS LAST,
         wl.duration_seconds ASC NULLS LAST
