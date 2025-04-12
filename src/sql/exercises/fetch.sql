SELECT e.*,
  (SELECT
    jsonb_agg(json_build_object(
        'id', s.id,
        'name', s.name,
        'duration', s.duration,
        'rep_count', s.rep_count,
        'rest_time', s.rest_time,
        'set_number', s.set_number,
        'created_at', s.created_at,
        'updated_at', s.updated_at
      ))
      FROM sets s
      WHERE s.exercise_id=e.id
  ) AS sets
FROM exercises e
WHERE id IN (?)