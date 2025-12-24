SELECT 
    u.*,
    (u.pro_until IS NOT NULL AND u.pro_until > NOW()) AS pro,
    row_to_json(m1.*) AS avatar
FROM users u
LEFT JOIN media m1 ON m1.id = u.avatar_id
WHERE u.id IN (?)
