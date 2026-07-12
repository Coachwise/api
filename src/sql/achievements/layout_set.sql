INSERT INTO profile_achievement_layouts (user_id, layout, updated_at)
VALUES ($1, $2, now())
ON CONFLICT (user_id) DO UPDATE SET layout = EXCLUDED.layout, updated_at = now()
RETURNING user_id
