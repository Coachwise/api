UPDATE notifications SET read = true WHERE user_id = $1 AND read = false
