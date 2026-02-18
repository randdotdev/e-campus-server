-- Tier limits (defaults per tier)
CREATE TABLE tier_limits (
    tier VARCHAR(50) PRIMARY KEY,
    max_colleges INT NOT NULL,
    max_departments_per_college INT NOT NULL,
    max_programs_per_department INT NOT NULL,
    max_students_per_program INT NOT NULL,
    max_applications_per_user INT NOT NULL,
    max_staff_users INT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Default tier limits
INSERT INTO tier_limits (tier, max_colleges, max_departments_per_college, max_programs_per_department, max_students_per_program, max_applications_per_user, max_staff_users) VALUES
('free', 3, 5, 5, 50, 2, 10),
('basic', 10, 20, 15, 300, 5, 100),
('premium', 100, 50, 30, 1000, 10, 500);

-- Current subscription (single row)
CREATE TABLE subscription (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tier VARCHAR(50) NOT NULL DEFAULT 'free' REFERENCES tier_limits(tier),
    -- Override columns (NULL = use tier default)
    max_colleges_override INT,
    max_departments_override INT,
    max_programs_override INT,
    max_students_override INT,
    max_applications_override INT,
    max_staff_override INT,
    expires_at TIMESTAMPTZ,
    updated_by UUID REFERENCES users(id),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Ensure only one subscription row
CREATE UNIQUE INDEX subscription_singleton ON subscription ((true));

-- Subscription history
CREATE TABLE subscription_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tier VARCHAR(50) NOT NULL,
    max_colleges_override INT,
    max_departments_override INT,
    max_programs_override INT,
    max_students_override INT,
    max_applications_override INT,
    max_staff_override INT,
    expires_at TIMESTAMPTZ,
    changed_by UUID REFERENCES users(id),
    changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    change_reason VARCHAR(255)
);

CREATE INDEX idx_subscription_history_changed_at ON subscription_history(changed_at DESC);

-- Insert default subscription
INSERT INTO subscription (tier) VALUES ('free');

-- Record initial state
INSERT INTO subscription_history (tier, change_reason) VALUES ('free', 'initial setup');
