-- Drop a token FCM reported dead, whoever owned it.
DELETE FROM device_tokens
WHERE token = $1;
