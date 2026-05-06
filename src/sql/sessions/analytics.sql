-- Get daily workout analytics with pagination
-- Aggregates all completed sessions by day
SELECT
    DATE(s.started_at)::text as date,
    COUNT(DISTINCT s.id) as sessions_count,
    SUM(CASE
        WHEN s.ended_at IS NOT NULL
        THEN EXTRACT(EPOCH FROM (s.ended_at - s.started_at)) / 60
        ELSE NULL
    END) as total_duration,
    array_agg(DISTINCT p.name) FILTER (WHERE p.name IS NOT NULL) as plans_completed,
    COUNT(DISTINCT wl.exercise_id) FILTER (WHERE wl.completed = true) as exercises_completed,
    COUNT(wl.id) FILTER (WHERE wl.completed = true) as total_sets,
    SUM(wl.reps) FILTER (WHERE wl.completed = true) as total_reps,
    SUM(wl.weight * wl.reps) FILTER (WHERE wl.completed = true) as total_volume,
    COUNT(*) OVER() as total_count
FROM sessions s
LEFT JOIN plans p ON p.id = s.plan_id
LEFT JOIN workout_logs wl ON wl.session_id = s.id
WHERE s.user_id = $1
GROUP BY DATE(s.started_at)
ORDER BY DATE(s.started_at) DESC
LIMIT $2 OFFSET $3
