SELECT user_id FROM chat_members WHERE chat_id = $1 AND user_id <> $2;
