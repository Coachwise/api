-- Includes deleted accounts. Only the OTP flow uses this: a deleted account must
-- still be able to receive a code, since verifying it is how the account is won
-- back. Everything else reads through users/fetch, which hides them.
SELECT * FROM users WHERE id = $1
