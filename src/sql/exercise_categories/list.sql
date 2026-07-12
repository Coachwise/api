SELECT id, slug, name_i18n, sport_type, sort_order, created_at, updated_at
FROM exercise_categories
WHERE ($1 = '' OR sport_type IS NULL OR sport_type::text = $1)
ORDER BY sort_order, slug;
