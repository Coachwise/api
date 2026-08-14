SELECT pe.*,
  COALESCE((SELECT jsonb_agg(json_build_object(
      'id', ps.id,
      'name', ps.name,
      'duration', ps.duration,
      'rep_count', ps.rep_count,
      'rest_time', ps.rest_time,
      'set_number', ps.set_number,
      'created_at', ps.created_at,
      'updated_at', ps.updated_at
    ) ORDER BY ps.set_number)
   FROM plan_exercise_sets ps WHERE ps.plan_exercise_id = pe.id), '[]'::jsonb) AS sets,
  json_build_object(
    'id', e.id,
    'name', e.name,
    'description', e.description,
    'name_i18n', e.name_i18n,
    'description_i18n', e.description_i18n,
    'sport_type', e.sport_type,
    -- A GROUP carries its rounds and its children so the runner can expand it
    -- into round -> exercise -> set without another call.
    'kind', e.kind,
    'rounds', e.rounds,
    'round_rest', e.round_rest,
    'round_duration', e.round_duration,
    'items', COALESCE((
      SELECT jsonb_agg(json_build_object(
          'id', it.id,
          'group_id', it.group_id,
          'exercise_id', it.exercise_id,
          'item_order', it.item_order,
          'rep_count', it.rep_count,
          'duration', it.duration,
          'rest_time', it.rest_time,
          'exercise', json_build_object(
            'id', ce.id,
            'name', ce.name,
            'name_i18n', ce.name_i18n,
            'description', ce.description,
            'description_i18n', ce.description_i18n,
            'sport_type', ce.sport_type,
            'kind', ce.kind,
            'public', ce.public,
            'user_id', ce.user_id,
            'media_id', ce.media_id,
            'track_weight', ce.track_weight,
            'track_distance', ce.track_distance,
            'track_grade', ce.track_grade,
            'track_height', ce.track_height,
            'sets', '[]'::jsonb,
            'created_at', ce.created_at,
            'updated_at', ce.updated_at,
            'media', CASE
              WHEN cm.id IS NOT NULL THEN
                json_build_object('id', cm.id, 'url', cm.url, 'filename', cm.filename, 'created_at', cm.created_at)
              ELSE NULL
            END
          )
        ) ORDER BY it.item_order)
      FROM exercise_items it
      JOIN exercises ce ON ce.id = it.exercise_id AND ce.deleted_at IS NULL
      LEFT JOIN media cm ON ce.media_id = cm.id
      WHERE it.group_id = e.id
    ), '[]'::jsonb),
    -- Without these the guided run can't know what to log and falls back to kg.
    'track_weight', e.track_weight,
    'track_distance', e.track_distance,
    'track_grade', e.track_grade,
    'track_height', e.track_height,
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
