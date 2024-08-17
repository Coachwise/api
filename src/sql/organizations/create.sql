INSERT INTO organizations (
  name, description, logo_id
) VALUES ( $1, $2, $3)
RETURNING *