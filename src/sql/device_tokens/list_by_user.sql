SELECT token, platform
FROM device_tokens
WHERE user_id = $1;
