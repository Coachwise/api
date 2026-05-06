SET search_path TO public;

-- Create enum type for plan schedule status
CREATE TYPE public.plan_schedule_status AS ENUM (
    'ACTIVE',
    'CANCELED'
);

-- Update existing data to match new enum values
-- SCHEDULED, PENDING, ACTIVE, COMPLETED -> ACTIVE
-- CANCELED -> CANCELED
UPDATE plan_schedules
SET status = CASE
    WHEN UPPER(status) IN ('SCHEDULED', 'PENDING', 'ACTIVE', 'COMPLETED') THEN 'ACTIVE'
    ELSE 'CANCELED'
END;

-- Drop the old default before changing column type
ALTER TABLE plan_schedules
    ALTER COLUMN status DROP DEFAULT;

-- Change column type to enum
ALTER TABLE plan_schedules
    ALTER COLUMN status TYPE public.plan_schedule_status
    USING status::public.plan_schedule_status;

-- Set new default value
ALTER TABLE plan_schedules
    ALTER COLUMN status SET DEFAULT 'ACTIVE'::public.plan_schedule_status;
