SELECT pe.*,
  json_build_object(
    'id', e.id,
    'name', e.name,
    'description', e.description,
    'sport_type', e.sport_type,
    'public', e.public,
    'user_id', e.user_id,
    'media_id', e.media_id,
    'created_at', e.created_at,
    'updated_at', e.updated_at,
    'media', CASE
      WHEN m.id IS NOT NULL THEN
        json_build_object(
          'id', m.id,
          'url', m.url,
          'filename', m.filename,
          'created_at', m.created_at
        )
      ELSE NULL
    END,
    'sets', (
      SELECT jsonb_agg(json_build_object(
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
      WHERE s.exercise_id = e.id
    )
  ) AS exercise
FROM plan_exercises pe
LEFT JOIN exercises e ON pe.exercise_id = e.id
LEFT JOIN media m ON e.media_id = m.id
WHERE pe.plan_id = $1
ORDER BY pe.exercise_order ASC
