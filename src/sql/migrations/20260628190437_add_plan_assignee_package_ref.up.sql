-- Track which package a plan assignment came from (NULL = assigned manually).
-- On unsubscribe we delete the package's assignments and keep the manual ones.
ALTER TABLE public.plan_assignees
    ADD COLUMN IF NOT EXISTS package_id uuid,
    ADD CONSTRAINT plan_assignees_package_fk FOREIGN KEY (package_id)
        REFERENCES public.coach_packages(id) ON DELETE SET NULL;

CREATE INDEX idx_plan_assignees_package ON public.plan_assignees (package_id);
