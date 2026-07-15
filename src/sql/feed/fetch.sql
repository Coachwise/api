SELECT *
FROM feeds
WHERE id IN (?) AND deleted_at IS NULL;
