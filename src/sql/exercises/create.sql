INSERT INTO exercises (user_id, name, description, public, sport_type, media_id)
VALUES ($1, $2, $3, $4,
    CASE WHEN $5::text = '' THEN 'GENERAL'::exercise_sport_type ELSE $5::exercise_sport_type END,
    $6)
RETURNING *