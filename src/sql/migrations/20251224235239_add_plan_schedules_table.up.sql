SET search_path TO public;

CREATE TABLE plan_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plan_id UUID NOT NULL REFERENCES plans(id) ON DELETE CASCADE,
    scheduled_for TIMESTAMPTZ NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING',
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_plan_schedules_user_id ON plan_schedules(user_id);
CREATE INDEX idx_plan_schedules_plan_id ON plan_schedules(plan_id);
CREATE INDEX idx_plan_schedules_scheduled_for ON plan_schedules(scheduled_for);
CREATE INDEX idx_plan_schedules_status ON plan_schedules(status);
