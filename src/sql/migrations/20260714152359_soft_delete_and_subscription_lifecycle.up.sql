-- Soft delete for the tables whose history we can't afford to lose, plus a real
-- lifecycle for package subscriptions.
--
-- Rows are never removed: deleting sets deleted_at, and every read filters on
-- `deleted_at IS NULL`. A refund, an audit or a dispute all need the row that a
-- DELETE would have destroyed.
--
-- Unique constraints are deliberately left alone. A deleted row keeps its
-- identity (email, phone, slug, package+client), and coming back REVIVES that
-- same row — an OTP login clears the user's deleted_at, re-subscribing clears the
-- subscription's. No duplicates to make room for, so no partial indexes.

-- 1) deleted_at everywhere it matters.
ALTER TABLE users                 ADD COLUMN IF NOT EXISTS deleted_at timestamptz;
ALTER TABLE exercises             ADD COLUMN IF NOT EXISTS deleted_at timestamptz;
ALTER TABLE plans                 ADD COLUMN IF NOT EXISTS deleted_at timestamptz;
ALTER TABLE coach_packages        ADD COLUMN IF NOT EXISTS deleted_at timestamptz;
ALTER TABLE package_subscriptions ADD COLUMN IF NOT EXISTS deleted_at timestamptz;
ALTER TABLE tests                 ADD COLUMN IF NOT EXISTS deleted_at timestamptz;
ALTER TABLE achievements          ADD COLUMN IF NOT EXISTS deleted_at timestamptz;
ALTER TABLE feeds                 ADD COLUMN IF NOT EXISTS deleted_at timestamptz;
ALTER TABLE workout_logs          ADD COLUMN IF NOT EXISTS deleted_at timestamptz;

-- Every read filters on deleted_at, so index for it.
CREATE INDEX IF NOT EXISTS users_live_idx                 ON users (id)                 WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS exercises_live_idx             ON exercises (id)             WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS plans_live_idx                 ON plans (id)                 WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS coach_packages_live_idx        ON coach_packages (id)        WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS package_subscriptions_live_idx ON package_subscriptions (id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS tests_live_idx                 ON tests (id)                 WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS achievements_live_idx          ON achievements (id)          WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS feeds_live_idx                 ON feeds (id)                 WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS workout_logs_live_idx          ON workout_logs (id)          WHERE deleted_at IS NULL;

-- 2) A subscription is a term, not a flag.
--
-- Nothing recorded when a subscription ended, so "the unused portion" — which is
-- what a refund is calculated from — was unknowable. ends_at fixes that; the
-- cancellation columns record who ended it and why, and order_id ties it back to
-- the money that paid for it.
ALTER TABLE package_subscriptions ADD COLUMN IF NOT EXISTS ends_at       timestamptz;
ALTER TABLE package_subscriptions ADD COLUMN IF NOT EXISTS canceled_at   timestamptz;
ALTER TABLE package_subscriptions ADD COLUMN IF NOT EXISTS canceled_by   uuid REFERENCES users (id);
ALTER TABLE package_subscriptions ADD COLUMN IF NOT EXISTS cancel_reason text;
ALTER TABLE package_subscriptions ADD COLUMN IF NOT EXISTS order_id      uuid REFERENCES orders (id);

-- Existing rows predate terms; give them one, so no row is left with a NULL
-- ends_at for the refund maths to trip over.
UPDATE package_subscriptions SET ends_at = created_at + interval '1 month' WHERE ends_at IS NULL;
