-- Logout: owner-scoped, so nobody can unregister someone else's device.
DELETE FROM device_tokens
WHERE token = $1 AND user_id = $2;
