-- Logging out an already-blacklisted token is a no-op, not an error: the token
-- is revoked either way, and a duplicate-key failure would otherwise count
-- against the database circuit breaker.
INSERT INTO tokens_blacklist (token)
VALUES ($1)
ON CONFLICT (token) DO NOTHING
