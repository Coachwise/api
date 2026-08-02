-- Register or re-register a device; an existing token moves to the new owner.
INSERT INTO device_tokens (user_id, token, platform, locale)
VALUES ($1, $2, $3, $4)
ON CONFLICT (token) DO UPDATE
SET user_id      = EXCLUDED.user_id,
    platform     = EXCLUDED.platform,
    locale       = EXCLUDED.locale,
    last_seen_at = now(),
    updated_at   = now()
RETURNING *;
