SELECT TRUE AS connected
FROM connections
WHERE user1_id = LEAST($1::uuid, $2::uuid) AND user2_id = GREATEST($1::uuid, $2::uuid)
LIMIT 1;
