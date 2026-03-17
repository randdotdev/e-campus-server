CREATE TABLE grading_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    offering_id UUID NOT NULL UNIQUE REFERENCES course_offerings(id) ON DELETE CASCADE,
    rules JSONB NOT NULL,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_grading_rules_offering ON grading_rules(offering_id);

CREATE TRIGGER update_grading_rules_updated_at
    BEFORE UPDATE ON grading_rules
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
