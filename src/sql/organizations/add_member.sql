INSERT INTO organization_members (
  user_id, organization_id
) VALUES ( $1, $2)
RETURNING *