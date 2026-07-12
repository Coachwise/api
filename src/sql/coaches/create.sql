INSERT INTO coaches (user_id, specialties)
VALUES ($1, ARRAY[$2]::sports[])
ON CONFLICT (user_id) DO NOTHING;
