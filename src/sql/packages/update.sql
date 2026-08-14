UPDATE coach_packages SET
    name = $2,
    description = $3,
    price_monthly = $4,
    price_quarterly = $5,
    price_annual = $6,
    price_one_time = $7,
    trial_days = $8,
    check_in_frequency = $9,
    video_access = $10,
    nutrition_guides = $11,
    custom_features = $12,
    is_active = $13,
    popular = $14,
    currency = $15,
    updated_at = now()
WHERE id = $1
RETURNING *
