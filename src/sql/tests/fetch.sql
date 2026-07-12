SELECT t.*,
       -- Minimal owner object (name + avatar). Built explicitly, not
       -- row_to_json(users), to avoid the password and non-_at timestamps like
       -- pro_until that the JSON hydrator can't parse.
       CASE WHEN c.id IS NOT NULL THEN jsonb_build_object(
           'id', c.id, 'username', c.username, 'first_name', c.first_name, 'last_name', c.last_name,
           'avatar', CASE WHEN cm.id IS NOT NULL
               THEN jsonb_build_object('id', cm.id, 'url', cm.url, 'filename', cm.filename)
               ELSE NULL END
       ) ELSE NULL END AS coach,
       (SELECT COUNT(*) FROM test_items ti WHERE ti.test_id = t.id) AS item_count,
       COALESCE((
           SELECT jsonb_agg(jsonb_build_object(
               'id', ti.id,
               'exercise_id', ti.exercise_id,
               'exercise_name', e.name,
               'track_reps', ti.track_reps, 'track_weight', ti.track_weight, 'track_time', ti.track_time,
               'target_value', ti.target_value,
               'item_order', ti.item_order
           ) ORDER BY ti.item_order)
           FROM test_items ti JOIN exercises e ON e.id = ti.exercise_id
           WHERE ti.test_id = t.id
       ), '[]'::jsonb) AS items
FROM tests t
LEFT JOIN users c ON c.id = t.coach_id
LEFT JOIN media cm ON cm.id = c.avatar_id
WHERE t.id IN (?)
