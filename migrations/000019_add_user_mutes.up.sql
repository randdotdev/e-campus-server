-- User mutes (course-level and university-wide)
CREATE TABLE user_mutes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    scope_type VARCHAR(20) NOT NULL CHECK (scope_type IN ('course', 'university')),
    scope_id UUID REFERENCES course_offerings(id) ON DELETE CASCADE,

    reason TEXT,

    muted_by UUID NOT NULL REFERENCES users(id),
    muted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ,

    unmuted_by UUID REFERENCES users(id),
    unmuted_at TIMESTAMPTZ,

    CONSTRAINT user_mutes_scope_check CHECK (
        (scope_type = 'university' AND scope_id IS NULL) OR
        (scope_type = 'course' AND scope_id IS NOT NULL)
    )
);

CREATE UNIQUE INDEX idx_user_mutes_active
    ON user_mutes(user_id, scope_type, scope_id)
    WHERE unmuted_at IS NULL;

CREATE INDEX idx_user_mutes_user ON user_mutes(user_id);
CREATE INDEX idx_user_mutes_offering ON user_mutes(scope_id) WHERE scope_id IS NOT NULL;
CREATE INDEX idx_user_mutes_muted_by ON user_mutes(muted_by);

-- Add 'course' to posts scope_type
ALTER TABLE posts DROP CONSTRAINT posts_scope_type_check;
ALTER TABLE posts ADD CONSTRAINT posts_scope_type_check
    CHECK (scope_type IN ('university', 'college', 'department', 'program', 'course'));

-- Update scope check to handle course scope
ALTER TABLE posts DROP CONSTRAINT posts_scope_check;
ALTER TABLE posts ADD CONSTRAINT posts_scope_check CHECK (
    (scope_type = 'university' AND scope_id IS NULL) OR
    (scope_type != 'university' AND scope_id IS NOT NULL)
);
