INSERT INTO connections (user1_id, user2_id, connection_request_id)
VALUES (LEAST($1::uuid, $2::uuid), GREATEST($1::uuid, $2::uuid), $3)
ON CONFLICT DO NOTHING
RETURNING *;
