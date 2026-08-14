INSERT INTO exercises (user_id, name, description, public, sport_type, media_id,
    track_weight, track_distance, track_grade, track_height,
    kind, rounds, round_rest, round_duration)
VALUES ($1, $2, $3, $4,
    CASE WHEN $5::text = '' THEN 'GENERAL'::exercise_sport_type ELSE $5::exercise_sport_type END,
    $6, $7, $8, $9, $10, $11, $12, $13, $14)
RETURNING *
