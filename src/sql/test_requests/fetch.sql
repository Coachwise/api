SELECT tr.*,
       row_to_json(a.*) AS athlete,
       row_to_json(c.*) AS coach,
       jsonb_build_object(
           'id', t.id,
           'name', COALESCE(t.name, tr.name),
           'description', t.description,
           'self', (tr.test_id IS NULL)
       ) AS test,
       COALESCE((
           SELECT jsonb_agg(jsonb_build_object(
               'id', ti.id,
               'exercise_id', ti.exercise_id,
               'exercise_name', e.name,
               'exercise_name_i18n', e.name_i18n,
               'track_reps', ti.track_reps, 'track_weight', ti.track_weight, 'track_time', ti.track_time,
               'target_value', ti.target_value,
               'item_order', ti.item_order
           ) ORDER BY ti.item_order)
           FROM test_items ti JOIN exercises e ON e.id = ti.exercise_id
           WHERE ti.test_id = tr.test_id
       ), '[]'::jsonb) AS items,
       COALESCE((
           SELECT jsonb_agg(jsonb_build_object(
               'exercise_id', wl.exercise_id,
               'exercise_name', ex.name,
               'exercise_name_i18n', ex.name_i18n,
               'reps', wl.reps,
               'weight', wl.weight,
               'duration_seconds', wl.duration_seconds
           ))
           FROM workout_logs wl JOIN exercises ex ON ex.id = wl.exercise_id
           WHERE wl.test_request_id = tr.id AND wl.deleted_at IS NULL
       ), '[]'::jsonb) AS records
FROM test_requests tr
LEFT JOIN tests t ON t.id = tr.test_id
LEFT JOIN users a ON a.id = tr.athlete_id
LEFT JOIN users c ON c.id = tr.coach_id
WHERE tr.id IN (?)
ORDER BY tr.created_at DESC
