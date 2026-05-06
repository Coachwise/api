INSERT INTO workout_logs_tags (workout_log_id, tag_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;
