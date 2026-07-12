DELETE FROM package_subscriptions WHERE package_id = $1 AND client_id = $2 RETURNING id
