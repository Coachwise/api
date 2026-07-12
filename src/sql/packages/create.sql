INSERT INTO coach_packages (
    coach_id, name, description,
    price_monthly, price_annual, price_one_time,
    trial_days, check_in_frequency, video_access, nutrition_guides,
    custom_features, is_active, popular
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
RETURNING *
