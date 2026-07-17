-- A SYSTEM line in the thread (a status marker, not user/admin prose). The caller
-- passes delivered_at: now() to keep the worker from pushing it (e.g. the user
-- closed their own ticket), or NULL to have the worker notify the user (e.g. the
-- admin closed it from the panel — but that path runs through Prisma, not this).
INSERT INTO support_messages (ticket_id, sender, body, delivered_at)
VALUES ($1, 'SYSTEM', $2, $3)
RETURNING *;
