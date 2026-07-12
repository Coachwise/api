SELECT n.*,
       -- Minimal actor object (name + avatar). Built explicitly, not
       -- row_to_json(users), to avoid the password + non-_at timestamps
       -- (pro_until) that the JSON hydrator can't parse.
       CASE WHEN a.id IS NOT NULL THEN jsonb_build_object(
           'id', a.id, 'username', a.username, 'first_name', a.first_name, 'last_name', a.last_name,
           'avatar', CASE WHEN am.id IS NOT NULL
               THEN jsonb_build_object('id', am.id, 'url', am.url, 'filename', am.filename)
               ELSE NULL END
       ) ELSE NULL END AS actor
FROM notifications n
LEFT JOIN users a ON a.id = n.actor_id
LEFT JOIN media am ON am.id = a.avatar_id
WHERE n.id IN (?)
ORDER BY n.created_at DESC
