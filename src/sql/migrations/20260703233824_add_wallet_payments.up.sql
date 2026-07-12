-- Wallet ledger + payments foundation.
-- Money is stored as whole-unit integers (Toman for IRR) as bigint, always with
-- a currency code, so the schema is ready for USD/Stripe later. Balances are
-- DERIVED from wallet_transactions (no cached column); escrow is time-based via
-- available_at. All statuses/types are varchar + CHECK for easy extension.

-- One wallet per (owner, currency). owner_id NULL = the platform system wallet
-- (holds fees + Pro revenue), one per currency.
CREATE TABLE public.wallets (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    owner_id uuid,
    currency character varying(3) NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    CONSTRAINT wallets_pkey PRIMARY KEY (id),
    CONSTRAINT wallets_owner_fk FOREIGN KEY (owner_id) REFERENCES public.users(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX wallets_owner_currency ON public.wallets (owner_id, currency) WHERE owner_id IS NOT NULL;
CREATE UNIQUE INDEX wallets_platform_currency ON public.wallets (currency) WHERE owner_id IS NULL;

-- The ledger. amount is signed (+credit / -debit). A credit counts toward the
-- available balance once available_at <= now(); until then it is "pending"
-- (escrow). Debits/payouts are always immediate.
CREATE TABLE public.wallet_transactions (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    wallet_id uuid NOT NULL,
    currency character varying(3) NOT NULL,
    amount bigint NOT NULL,
    type character varying(16) NOT NULL,
    available_at timestamp without time zone DEFAULT now() NOT NULL,
    ref_type character varying(24),
    ref_id uuid,
    description text,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    CONSTRAINT wallet_transactions_pkey PRIMARY KEY (id),
    CONSTRAINT wallet_transactions_wallet_fk FOREIGN KEY (wallet_id) REFERENCES public.wallets(id) ON DELETE CASCADE,
    CONSTRAINT wallet_transactions_type_chk CHECK (type IN ('TOPUP','PURCHASE','SALE','FEE','PAYOUT','REFUND'))
);
CREATE INDEX idx_wallet_tx_wallet ON public.wallet_transactions (wallet_id, available_at);
CREATE INDEX idx_wallet_tx_ref ON public.wallet_transactions (ref_type, ref_id);

-- A purchase (Pro or a coach package). The audit record tying ledger entries
-- together. All amounts are the currency's whole unit.
CREATE TABLE public.orders (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    buyer_id uuid NOT NULL,
    kind character varying(16) NOT NULL,
    currency character varying(3) NOT NULL,
    coach_id uuid,
    package_id uuid,
    duration_months integer NOT NULL,
    unit_amount bigint NOT NULL,
    subtotal bigint NOT NULL,
    discount_amount bigint NOT NULL DEFAULT 0,
    fee_amount bigint NOT NULL DEFAULT 0,
    total bigint NOT NULL,
    status character varying(16) NOT NULL DEFAULT 'PENDING',
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    CONSTRAINT orders_pkey PRIMARY KEY (id),
    CONSTRAINT orders_buyer_fk FOREIGN KEY (buyer_id) REFERENCES public.users(id) ON DELETE CASCADE,
    CONSTRAINT orders_coach_fk FOREIGN KEY (coach_id) REFERENCES public.users(id) ON DELETE SET NULL,
    CONSTRAINT orders_package_fk FOREIGN KEY (package_id) REFERENCES public.coach_packages(id) ON DELETE SET NULL,
    CONSTRAINT orders_kind_chk CHECK (kind IN ('PRO','PACKAGE')),
    CONSTRAINT orders_status_chk CHECK (status IN ('PENDING','PAID','CANCELED','REFUNDED'))
);
CREATE INDEX idx_orders_buyer ON public.orders (buyer_id, created_at DESC);
CREATE INDEX idx_orders_coach ON public.orders (coach_id, created_at DESC);

-- Money-in events (top-ups). The stub provider marks them PAID immediately; real
-- gateways implement the same shape later.
CREATE TABLE public.payments (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    order_id uuid,
    user_id uuid NOT NULL,
    wallet_id uuid NOT NULL,
    amount bigint NOT NULL,
    currency character varying(3) NOT NULL,
    provider character varying(24) NOT NULL DEFAULT 'STUB',
    provider_ref character varying(128),
    status character varying(16) NOT NULL DEFAULT 'PENDING',
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    CONSTRAINT payments_pkey PRIMARY KEY (id),
    CONSTRAINT payments_order_fk FOREIGN KEY (order_id) REFERENCES public.orders(id) ON DELETE SET NULL,
    CONSTRAINT payments_user_fk FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE,
    CONSTRAINT payments_wallet_fk FOREIGN KEY (wallet_id) REFERENCES public.wallets(id) ON DELETE CASCADE,
    CONSTRAINT payments_status_chk CHECK (status IN ('PENDING','PAID','FAILED'))
);
CREATE INDEX idx_payments_user ON public.payments (user_id, created_at DESC);

-- Coach withdrawal (top-out) requests. Draws from the available balance. Admin
-- approve/mark-paid transitions arrive with the future admin dashboard.
CREATE TABLE public.payouts (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    coach_id uuid NOT NULL,
    wallet_id uuid NOT NULL,
    amount bigint NOT NULL,
    currency character varying(3) NOT NULL,
    status character varying(16) NOT NULL DEFAULT 'REQUESTED',
    note text,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    CONSTRAINT payouts_pkey PRIMARY KEY (id),
    CONSTRAINT payouts_coach_fk FOREIGN KEY (coach_id) REFERENCES public.users(id) ON DELETE CASCADE,
    CONSTRAINT payouts_wallet_fk FOREIGN KEY (wallet_id) REFERENCES public.wallets(id) ON DELETE CASCADE,
    CONSTRAINT payouts_status_chk CHECK (status IN ('REQUESTED','APPROVED','PAID','REJECTED'))
);
CREATE INDEX idx_payouts_coach ON public.payouts (coach_id, created_at DESC);

-- ---- Pricing config (DB-backed; an admin dashboard can edit these later) ----

-- Single-row platform settings.
CREATE TABLE public.platform_settings (
    id smallint PRIMARY KEY DEFAULT 1,
    coach_fee_percent integer NOT NULL DEFAULT 15,
    escrow_hold_days integer NOT NULL DEFAULT 0,
    default_currency character varying(3) NOT NULL DEFAULT 'IRR',
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    CONSTRAINT platform_settings_singleton CHECK (id = 1)
);

-- Pro membership monthly price, per currency.
CREATE TABLE public.pro_prices (
    currency character varying(3) PRIMARY KEY,
    monthly_amount bigint NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL
);

-- Shared duration discount tiers (apply to both Pro and coach packages).
CREATE TABLE public.duration_tiers (
    months integer PRIMARY KEY,
    discount_percent integer NOT NULL DEFAULT 0
);

-- Coach package prices are the currency's whole unit; tag the currency.
ALTER TABLE public.coach_packages ADD COLUMN currency character varying(3) NOT NULL DEFAULT 'IRR';

-- ---- Seed defaults ----
INSERT INTO public.platform_settings (id, coach_fee_percent, escrow_hold_days, default_currency)
    VALUES (1, 15, 0, 'IRR') ON CONFLICT (id) DO NOTHING;
INSERT INTO public.pro_prices (currency, monthly_amount)
    VALUES ('IRR', 99000) ON CONFLICT (currency) DO NOTHING;
INSERT INTO public.duration_tiers (months, discount_percent)
    VALUES (1, 0), (3, 10), (12, 20) ON CONFLICT (months) DO NOTHING;
INSERT INTO public.wallets (owner_id, currency) VALUES (NULL, 'IRR');
