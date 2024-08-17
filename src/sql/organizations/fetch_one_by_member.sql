SELECT o.* FROM organizations o
JOIN organization_members m ON user_id=$2 AND m.organization_id=o.id
WHERE o.id=$1