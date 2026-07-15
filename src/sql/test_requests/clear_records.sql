-- Re-submitting an assessment replaces its records; the old ones stay for history.
UPDATE workout_logs SET deleted_at = now()
WHERE test_request_id = $1 AND deleted_at IS NULL
