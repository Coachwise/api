SELECT currency, amount FROM coach_package_prices WHERE package_id = $1 ORDER BY currency;
