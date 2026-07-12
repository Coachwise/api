-- A coach's payout destination, one per (user, currency). The method is chosen
-- by currency: IRR pays out to a bank CARD; other currencies will use STRIPE
-- (Stripe Connect) once wired, or a BANK account-to-account transfer meanwhile.
-- Columns for every method exist up front so adding Stripe needs no migration.
CREATE TABLE public.payout_accounts (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    user_id uuid NOT NULL,
    currency character varying(3) NOT NULL,
    method character varying(16) NOT NULL,
    account_holder text,
    card_number text,
    iban text,
    bank_name text,
    swift text,
    stripe_account_id text,
    status character varying(16) NOT NULL DEFAULT 'UNVERIFIED',
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    CONSTRAINT payout_accounts_pkey PRIMARY KEY (id),
    CONSTRAINT payout_accounts_user_fk FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE,
    CONSTRAINT payout_accounts_currency_fk FOREIGN KEY (currency) REFERENCES public.currencies(code),
    CONSTRAINT payout_accounts_method_chk CHECK (method IN ('CARD','BANK','STRIPE')),
    CONSTRAINT payout_accounts_status_chk CHECK (status IN ('UNVERIFIED','VERIFIED')),
    CONSTRAINT payout_accounts_user_currency_uniq UNIQUE (user_id, currency)
);
CREATE INDEX idx_payout_accounts_user ON public.payout_accounts (user_id);
