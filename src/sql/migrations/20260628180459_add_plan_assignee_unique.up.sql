-- A plan should be assigned to a user at most once. Remove any existing
-- duplicates (keep one row per plan+user), then enforce uniqueness.
DELETE FROM public.plan_assignees a
USING public.plan_assignees b
WHERE a.ctid < b.ctid
  AND a.plan_id = b.plan_id
  AND a.user_id = b.user_id;

ALTER TABLE public.plan_assignees
    ADD CONSTRAINT plan_assignees_plan_user_unique UNIQUE (plan_id, user_id);
