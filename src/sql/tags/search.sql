SELECT
    t.id,
    COUNT(*) OVER () as total_count
FROM tags t
LEFT JOIN workout_logs_tags wlt ON t.id = wlt.tag_id
LEFT JOIN workout_logs wl ON wlt.workout_log_id = wl.id AND wl.deleted_at IS NULL
LEFT JOIN sessions s ON wl.session_id = s.id
WHERE
    ($1 = '' OR t.search_vector @@ to_tsquery('english',
        (SELECT string_agg(word || ':*', ' & ')
         FROM unnest(regexp_split_to_array($1, E'\\s+')) AS word
         WHERE word != '')
    ))
    AND (COALESCE(NULLIF($2, ''), s.session_type) = s.session_type OR $2 = '')
GROUP BY t.id
ORDER BY
    CASE WHEN $1 != '' THEN ts_rank(t.search_vector, to_tsquery('english',
        (SELECT string_agg(word || ':*', ' & ')
         FROM unnest(regexp_split_to_array($1, E'\\s+')) AS word
         WHERE word != '')
    )) END DESC,
    COUNT(wlt.workout_log_id) DESC,
    t.created_at DESC
LIMIT $3 OFFSET $4;
