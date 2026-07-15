-- Refunds need to know how the buyer's money was split, and orders didn't record
-- it: fee_amount was stored, but the coach's net and the Pro portion were only
-- ever computed in the quote. Without them a refund can't take back the right
-- amount from the right wallet.

ALTER TABLE orders ADD COLUMN IF NOT EXISTS coach_net       bigint NOT NULL DEFAULT 0;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS pro_amount      bigint NOT NULL DEFAULT 0;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS refunded_amount bigint NOT NULL DEFAULT 0;

-- Orders written before this migration: everything that wasn't the platform fee
-- went to the coach, and no Pro was itemised separately.
UPDATE orders SET coach_net = total - fee_amount
WHERE kind = 'PACKAGE' AND coach_net = 0;

-- A refund can be partial: the part of the term the client already used is not
-- given back. 'PARTIALLY_REFUNDED' is 18 characters and the column held 16.
ALTER TABLE orders ALTER COLUMN status TYPE character varying(24);
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_status_chk;
ALTER TABLE orders ADD CONSTRAINT orders_status_chk
    CHECK (status IN ('PENDING','PAID','CANCELED','REFUNDED','PARTIALLY_REFUNDED'));

-- The cooling-off window is the escrow: while the coach's earnings are still
-- held, a cancellation costs nobody anything to undo. It was 0 days, which meant
-- earnings were available instantly and there was no window at all.
UPDATE platform_settings SET escrow_hold_days = 7 WHERE escrow_hold_days = 0;
