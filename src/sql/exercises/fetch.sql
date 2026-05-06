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
  ) AS sets,
  CASE
    WHEN m.id IS NOT NULL THEN
      json_build_object(
        'id', m.id,
        'url', m.url,
        'filename', m.filename,
        'created_at', m.created_at
      )
    ELSE NULL
  END AS media
FROM exercises e
LEFT JOIN media m ON e.media_id = m.id
WHERE e.id IN (?)