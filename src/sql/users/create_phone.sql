-- Passwordless phone signup: no password, a placeholder email (never used for
-- login), an auto username. Starts INACTIVE until the first OTP is verified.
INSERT INTO users (username, email, phone, status)
VALUES ($1, $2, $3, 'INACTIVE')
RETURNING *
