-- One row per (protocol, client) the coach has assigned, with the client and
-- their run stats — mirrors the athlete's assigned list from the coach's side.
SELECT x.* FROM (
    SELECT DISTINCT ON (asg.test_id, asg.athlete_id)
           asg.test_id,
           asg.athlete_id,
           asg.created_at AS assigned_at,
           t.name AS test_name,
           (SELECT count(*) FROM test_items ti WHERE ti.test_id = asg.test_id) AS item_count,
           -- Minimal client object (name + avatar). Built explicitly rather than
           -- row_to_json(users) to avoid dumping the password and non-_at
           -- timestamps like pro_until (which the JSON hydrator can't parse).
           jsonb_build_object(
               'id', a.id,
               'username', a.username,
               'first_name', a.first_name,
               'last_name', a.last_name,
               'avatar', CASE WHEN m.id IS NOT NULL
                   THEN jsonb_build_object('id', m.id, 'url', m.url, 'filename', m.filename)
                   ELSE NULL END
           ) AS athlete,
           (SELECT count(*) FROM test_requests r
              WHERE r.test_id = asg.test_id AND r.athlete_id = asg.athlete_id AND r.status <> 'PENDING') AS runs_count,
           (SELECT max(COALESCE(r.submitted_at, r.created_at)) FROM test_requests r
              WHERE r.test_id = asg.test_id AND r.athlete_id = asg.athlete_id AND r.status <> 'PENDING') AS last_run_at
    FROM test_requests asg
    JOIN tests t ON t.id = asg.test_id
    JOIN users a ON a.id = asg.athlete_id
    LEFT JOIN media m ON m.id = a.avatar_id
    WHERE asg.coach_id = $1
    ORDER BY asg.test_id, asg.athlete_id, asg.created_at DESC
) x
ORDER BY x.last_run_at DESC NULLS LAST, x.assigned_at DESC
