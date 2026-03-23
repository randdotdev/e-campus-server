-- User preferences
CREATE TABLE user_preferences (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    language VARCHAR(10) NOT NULL DEFAULT 'en',
    timezone VARCHAR(50) NOT NULL DEFAULT 'UTC',
    email_notifications BOOLEAN NOT NULL DEFAULT true,
    push_notifications BOOLEAN NOT NULL DEFAULT true,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Restructure settings table with full university configuration
-- Preserves existing full_year_repeat value if present
UPDATE settings SET settings = jsonb_build_object(
    'institution', jsonb_build_object(
        'name_en', 'University',
        'name_ku', '',
        'name_ar', '',
        'type', 'public',
        'country', 'Iraq',
        'region', 'Kurdistan'
    ),
    'degree_labels', jsonb_build_object(
        'bachelor', jsonb_build_object('en', 'Bachelor', 'local', ''),
        'master', jsonb_build_object('en', 'Master', 'local', ''),
        'phd', jsonb_build_object('en', 'Doctorate', 'local', '')
    ),
    'grading', jsonb_build_object(
        'display', 'numeric',
        'scale', jsonb_build_object('A', 90, 'B', 80, 'C', 70, 'D', 60, 'E', 50, 'F', 0)
    ),
    'features', jsonb_build_object(
        'credits_tracking', true,
        'allow_retake', true,
        'allow_pretake', true,
        'full_year_repeat', COALESCE((settings->>'full_year_repeat')::boolean, false),
        'grade_visibility', true
    ),
    'academic', jsonb_build_object(
        'semesters_per_year', 2,
        'max_failure_repeats', 2,
        'default_language', 'en'
    )
);
