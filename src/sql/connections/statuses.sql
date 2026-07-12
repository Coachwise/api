-- Per-viewer connection state for every user the viewer has any relationship
-- with. Postgres computes the status string and boolean; one row per other user
-- (DISTINCT ON + priority: connected > pending_incoming > pending_outgoing).
SELECT DISTINCT ON (id) id, is_connected, connection_status
FROM (
    SELECT CASE WHEN user1_id = $1 THEN user2_id ELSE user1_id END AS id,
           TRUE  AS is_connected,
           'connected' AS connection_status,
           1 AS priority
    FROM connections
    WHERE user1_id = $1 OR user2_id = $1

    UNION ALL

    SELECT requester_id AS id,
           FALSE AS is_connected,
           'pending_incoming' AS connection_status,
           2 AS priority
    FROM connection_requests
    WHERE addressee_id = $1 AND status = 'PENDING'

    UNION ALL

    -- The requester always sees a request as pending (soft reject), so every
    -- status the viewer sent counts as pending_outgoing.
    SELECT addressee_id AS id,
           FALSE AS is_connected,
           'pending_outgoing' AS connection_status,
           3 AS priority
    FROM connection_requests
    WHERE requester_id = $1
) t
ORDER BY id, priority;
