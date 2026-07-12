-- Multi-currency prices + sale type for coach packages.
--
-- A package is EITHER a subscription (fixed monthly price; buyer picks 1/3/12
-- months with the shared duration tiers) OR a one-time purchase (pay once; the
-- bundled plans are lifetime in the buyer's profile). Both grant Pro:
-- subscriptions for the chosen months, one-time for platform_settings
-- .one_time_pro_months.
ALTER TABLE public.coach_packages
    ADD COLUMN billing_type character varying(16) NOT NULL DEFAULT 'SUBSCRIPTION',
    ADD CONSTRAINT coach_packages_billing_type_chk CHECK (billing_type IN ('SUBSCRIPTION','ONE_TIME'));

-- Per-currency price: the source of truth for which currencies a package sells
-- in. `amount` is the currency's whole unit (Toman for IRR) — the monthly price
-- for subscriptions, the flat price for one-time. coach_packages.currency stays
-- as the default/display currency. No FX conversion — coach sets each price.
CREATE TABLE public.coach_package_prices (
    package_id uuid NOT NULL,
    currency character varying(3) NOT NULL,
    amount bigint NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    CONSTRAINT coach_package_prices_pkey PRIMARY KEY (package_id, currency),
    CONSTRAINT coach_package_prices_package_fk FOREIGN KEY (package_id) REFERENCES public.coach_packages(id) ON DELETE CASCADE
);

-- How many months of Pro a one-time purchase grants (admin-tunable later).
ALTER TABLE public.platform_settings
    ADD COLUMN one_time_pro_months integer NOT NULL DEFAULT 1;

-- Backfill: seed each package's default-currency price from its existing
-- price_monthly so current packages stay purchasable.
INSERT INTO public.coach_package_prices (package_id, currency, amount)
SELECT id, currency, price_monthly
FROM public.coach_packages
WHERE price_monthly IS NOT NULL
ON CONFLICT (package_id, currency) DO NOTHING;
