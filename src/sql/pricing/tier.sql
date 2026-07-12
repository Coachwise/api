SELECT months, discount_percent
FROM duration_tiers
WHERE months = $1;
