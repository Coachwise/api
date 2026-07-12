-- Platform-supported currencies. The single source of truth for which currency
-- codes are accepted anywhere money is priced or moved. `decimals` is the number
-- of minor-unit digits (0 for Toman — amounts are whole Toman). `enabled` lets an
-- admin turn a currency off without deleting its history.
CREATE TABLE public.currencies (
    code character varying(3) NOT NULL,
    name character varying(64) NOT NULL,
    symbol character varying(8) NOT NULL,
    decimals smallint NOT NULL DEFAULT 0,
    enabled boolean NOT NULL DEFAULT true,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    CONSTRAINT currencies_pkey PRIMARY KEY (code)
);

-- Seed the launch currency (Iran). Amounts are stored in whole Toman.
INSERT INTO public.currencies (code, name, symbol, decimals, enabled)
VALUES ('IRR', 'Toman', 'تومان', 0, true)
ON CONFLICT (code) DO NOTHING;

-- Integrity: prices/config may only reference a known currency.
ALTER TABLE public.pro_prices
    ADD CONSTRAINT pro_prices_currency_fk FOREIGN KEY (currency) REFERENCES public.currencies(code);
ALTER TABLE public.coach_package_prices
    ADD CONSTRAINT coach_package_prices_currency_fk FOREIGN KEY (currency) REFERENCES public.currencies(code);
