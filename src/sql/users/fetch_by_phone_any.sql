-- Includes deleted accounts, unlike fetch_by_phone. Only the OTP flow uses this:
-- a deleted account must still be found (so the code goes to the right row and no
-- duplicate is created), but it is not revived until the code is verified.
SELECT * FROM users WHERE phone = $1 LIMIT 1
