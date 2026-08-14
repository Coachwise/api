UPDATE exercises SET
    name=$2,
    description=$3,
    public=$4,
    sport_type=CASE WHEN $5::text = '' THEN sport_type ELSE $5::exercise_sport_type END,
    media_id=$6,
    track_weight=$7,
    track_distance=$8,
    track_grade=$9,
    track_height=$10,
    kind=$11,
    rounds=$12,
    round_rest=$13,
    round_duration=$14,
    updated_at=NOW()
WHERE id=$1 AND deleted_at IS NULL
RETURNING *
