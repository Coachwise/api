INSERT INTO coach_applications
  (user_id, full_name, specialty, experience_years, certifications, bio, website, instagram, decision_token)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;
