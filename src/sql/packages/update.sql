UPDATE coach_packages SET
    name = $2,
    description = $3,
    price_monthly = $4,
    price_annual = $5,
    price_one_time = $6,
    trial_days = $7,
    check_in_frequency = $8,
    video_access = $9,
    nutrition_guides = $10,
    custom_features = $11,
    is_active = $12,
    popular = $13,
    updated_at = now()
WHERE id = $1
RETURNING *
