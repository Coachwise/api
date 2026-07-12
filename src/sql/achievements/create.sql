INSERT INTO achievements (athlete_id, coach_id, title, description)
VALUES ($1, $2, $3, $4)
RETURNING *
