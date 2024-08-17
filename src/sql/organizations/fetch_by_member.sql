SELECT o.* FROM organizations o
JOIN organization_members m ON user_id=$1 AND m.organization_id=o.id
LIMIT 10