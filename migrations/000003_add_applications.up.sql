CREATE TABLE applications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Applicant (SET NULL preserves application history if user deleted)
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,

    -- Target program (RESTRICT prevents deleting program with applications)
    program_id UUID NOT NULL REFERENCES programs(id) ON DELETE RESTRICT,

    -- Admission details
    admission_year INT NOT NULL CHECK (admission_year BETWEEN 2000 AND 2100),
    study_type VARCHAR(20) NOT NULL CHECK (study_type IN ('morning', 'evening')),

    -- Universal applicant data (columns for querying/filtering/reporting)
    date_of_birth DATE NOT NULL,
    gender VARCHAR(10) NOT NULL CHECK (gender IN ('male', 'female', 'other')),
    nationality VARCHAR(100) NOT NULL,

    -- Flexible data (varies by institution)
    personal_extra JSONB NOT NULL DEFAULT '{}',
    academic JSONB NOT NULL DEFAULT '{}',
    documents JSONB NOT NULL DEFAULT '[]',

    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'approved', 'rejected', 'withdrawn', 'needs_revision')),

    -- Review (must be consistent: both set or both null)
    reviewed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at TIMESTAMPTZ,
    review_notes TEXT,

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Constraints
    CHECK (
        (reviewed_by IS NULL AND reviewed_at IS NULL) OR
        (reviewed_by IS NOT NULL AND reviewed_at IS NOT NULL)
    )
);

-- Indexes for common queries
CREATE INDEX idx_applications_user ON applications(user_id);
CREATE INDEX idx_applications_program ON applications(program_id);
CREATE INDEX idx_applications_status ON applications(status);
CREATE INDEX idx_applications_admission_year ON applications(admission_year);

-- Cursor pagination
CREATE INDEX idx_applications_created ON applications(created_at DESC, id DESC);

-- Filtering indexes
CREATE INDEX idx_applications_nationality ON applications(nationality);
CREATE INDEX idx_applications_gender ON applications(gender);

-- Prevent duplicate pending/needs_revision applications per user per program per year
CREATE UNIQUE INDEX idx_applications_pending_unique
ON applications(user_id, program_id, admission_year)
WHERE status IN ('pending', 'needs_revision');

-- updated_at trigger
CREATE TRIGGER update_applications_updated_at
BEFORE UPDATE ON applications
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
