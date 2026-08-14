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
  END AS media,
  -- A GROUP's children in round order, each with the exercise it points at so
  -- the runner has names/media/track flags without a second call. Children are
  -- always SINGLE (one level), so this can't recurse.
  COALESCE((SELECT
    jsonb_agg(json_build_object(
        'id', it.id,
        'group_id', it.group_id,
        'exercise_id', it.exercise_id,
        'item_order', it.item_order,
        'rep_count', it.rep_count,
        'duration', it.duration,
        'rest_time', it.rest_time,
        'created_at', it.created_at,
        'updated_at', it.updated_at,
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
  ), '[]'::jsonb) AS items
FROM exercises e
LEFT JOIN media m ON e.media_id = m.id
WHERE e.id IN (?) AND e.deleted_at IS NULL
-- Keep the order list.sql chose (yours first, then relevance, then newest);
-- without this the rows come back in planner order.
ORDER BY array_position(ARRAY[?]::uuid[], e.id)