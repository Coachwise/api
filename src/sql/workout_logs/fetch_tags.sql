SELECT
    t.id,
    COUNT(*) OVER () AS total_count
FROM workout_logs_tags wlt
JOIN tags t ON wlt.tag_id = t.id
WHERE wlt.workout_log_id = $1;
