-- Coach packages: a coach-owned, sellable "tier" that bundles a set of plans.
-- Pricing/features are stored as metadata only for now (no payment processing).
CREATE TABLE public.coach_packages (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    coach_id uuid NOT NULL,
    name character varying(128) NOT NULL,
    description text,
    -- Prices are whole-unit integers (Toman, no decimals); NULL = not offered.
    price_monthly integer,
    price_annual integer,
    price_one_time integer,
    trial_days integer NOT NULL DEFAULT 0,
    check_in_frequency character varying(32),
    video_access boolean NOT NULL DEFAULT false,
    nutrition_guides boolean NOT NULL DEFAULT false,
    custom_features jsonb NOT NULL DEFAULT '[]'::jsonb,
    is_active boolean NOT NULL DEFAULT true,
    popular boolean NOT NULL DEFAULT false,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    CONSTRAINT coach_packages_pkey PRIMARY KEY (id),
    CONSTRAINT coach_packages_coach_fk FOREIGN KEY (coach_id) REFERENCES public.users(id) ON DELETE CASCADE
);

CREATE INDEX idx_coach_packages_coach ON public.coach_packages (coach_id, created_at DESC);
CREATE INDEX idx_coach_packages_active ON public.coach_packages (coach_id) WHERE is_active;

-- Junction: which plans are bundled into a package.
CREATE TABLE public.coach_package_plans (
    package_id uuid NOT NULL,
    plan_id uuid NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    CONSTRAINT coach_package_plans_pkey PRIMARY KEY (package_id, plan_id),
    CONSTRAINT coach_package_plans_package_fk FOREIGN KEY (package_id) REFERENCES public.coach_packages(id) ON DELETE CASCADE,
    CONSTRAINT coach_package_plans_plan_fk FOREIGN KEY (plan_id) REFERENCES public.plans(id) ON DELETE CASCADE
);

CREATE INDEX idx_coach_package_plans_plan ON public.coach_package_plans (plan_id);
