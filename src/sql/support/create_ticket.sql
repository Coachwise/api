-- A new ticket opens already awaiting the admin (turn defaults to ADMIN), because
-- creation carries the user's first message.
INSERT INTO support_tickets (user_id, subject)
VALUES ($1, $2)
RETURNING *;
