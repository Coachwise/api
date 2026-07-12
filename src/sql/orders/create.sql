INSERT INTO orders
    (buyer_id, kind, currency, coach_id, package_id, duration_months,
     unit_amount, subtotal, discount_amount, fee_amount, total, status)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING *;
