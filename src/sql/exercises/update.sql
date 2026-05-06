UPDATE exercises SET
    name=$2,
    description=$3,
    public=$4,
    sport_type=CASE WHEN $5::text = '' THEN sport_type ELSE $5::exercise_sport_type END,
    media_id=$6,
    updated_at=NOW()
WHERE id=$1
RETURNING *