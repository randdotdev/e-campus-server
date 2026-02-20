-- Students (created when application approved)
CREATE TABLE students (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    program_id UUID NOT NULL REFERENCES programs(id) ON DELETE RESTRICT,

    admission_year INT NOT NULL CHECK (admission_year >= 2000 AND admission_year <= 2100),
    current_cohort_year INT NOT NULL CHECK (current_cohort_year >= 2000 AND current_cohort_year <= 2100),
    current_year INT NOT NULL DEFAULT 1 CHECK (current_year >= 1 AND current_year <= 8),

    shift VARCHAR(10) NOT NULL CHECK (shift IN ('day', 'evening')),
    tuition VARCHAR(10) NOT NULL CHECK (tuition IN ('free', 'paid')),

    status VARCHAR(20) NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'graduated', 'withdrawn', 'suspended', 'on_leave')),

    enrolled_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CHECK (current_cohort_year >= admission_year)
);

CREATE INDEX idx_students_user ON students(user_id);
CREATE INDEX idx_students_program ON students(program_id);
CREATE INDEX idx_students_cohort ON students(current_cohort_year);
CREATE INDEX idx_students_status ON students(status);

-- Student Leaves (tracks leave periods)
CREATE TABLE student_leaves (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    reason VARCHAR(50) NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE,
    approved_by UUID REFERENCES users(id) ON DELETE SET NULL,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_student_leaves_student ON student_leaves(student_id);

-- Student Cohort History (tracks cohort/stage changes)
CREATE TABLE student_cohort_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    from_cohort_year INT NOT NULL,
    to_cohort_year INT NOT NULL,
    from_year INT NOT NULL,
    to_year INT NOT NULL,
    reason VARCHAR(20) NOT NULL CHECK (reason IN ('failed', 'transferred', 'returned')),
    notes TEXT,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_cohort_history_student ON student_cohort_history(student_id);

-- Program Curriculum (courses per program/cohort/year/semester)
CREATE TABLE program_curriculum (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_id UUID NOT NULL REFERENCES programs(id) ON DELETE CASCADE,
    cohort_year INT NOT NULL CHECK (cohort_year >= 2000 AND cohort_year <= 2100),
    year INT NOT NULL CHECK (year >= 1 AND year <= 8),
    semester VARCHAR(10) NOT NULL CHECK (semester IN ('fall', 'spring')),
    course_id UUID NOT NULL REFERENCES courses(id) ON DELETE RESTRICT,
    is_required BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(program_id, cohort_year, year, semester, course_id)
);

CREATE INDEX idx_curriculum_program ON program_curriculum(program_id);
CREATE INDEX idx_curriculum_cohort ON program_curriculum(program_id, cohort_year);
CREATE INDEX idx_curriculum_course ON program_curriculum(course_id);

-- Semester Requirements (ECTS thresholds, tracked historically)
CREATE TABLE semester_requirements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_id UUID NOT NULL REFERENCES programs(id) ON DELETE CASCADE,
    cohort_year INT NOT NULL CHECK (cohort_year >= 2000 AND cohort_year <= 2100),
    year INT NOT NULL CHECK (year >= 1 AND year <= 8),
    semester VARCHAR(10) NOT NULL CHECK (semester IN ('fall', 'spring')),
    min_ects INT NOT NULL CHECK (min_ects > 0),
    effective_from DATE NOT NULL,
    effective_to DATE,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CHECK (effective_to IS NULL OR effective_to > effective_from)
);

CREATE INDEX idx_semester_req_program ON semester_requirements(program_id);
CREATE INDEX idx_semester_req_lookup ON semester_requirements(program_id, cohort_year, year, semester);

-- Prevent overlapping effective dates for same program/cohort/year/semester
CREATE EXTENSION IF NOT EXISTS btree_gist;

ALTER TABLE semester_requirements
    ADD CONSTRAINT semester_requirements_no_overlap
    EXCLUDE USING gist (
        program_id WITH =,
        cohort_year WITH =,
        year WITH =,
        (semester::text) WITH =,
        daterange(effective_from, COALESCE(effective_to, '9999-12-31'::date), '[]') WITH &&
    );

-- Payments (for paid students)
CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    semester_id UUID NOT NULL REFERENCES semesters(id) ON DELETE RESTRICT,

    amount DECIMAL(10,2) NOT NULL CHECK (amount >= 0),
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',

    status VARCHAR(20) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'paid', 'overdue', 'waived')),

    due_date DATE NOT NULL,
    paid_at TIMESTAMPTZ,

    receipt_number VARCHAR(100),
    notes TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CHECK (
        (status != 'paid' AND paid_at IS NULL) OR
        (status = 'paid' AND paid_at IS NOT NULL) OR
        (status = 'waived')
    )
);

CREATE INDEX idx_payments_student ON payments(student_id);
CREATE INDEX idx_payments_semester ON payments(semester_id);
CREATE INDEX idx_payments_status ON payments(status);

-- Update applications: study_type -> shift + tuition
ALTER TABLE applications ADD COLUMN shift VARCHAR(10);
ALTER TABLE applications ADD COLUMN tuition VARCHAR(10);

UPDATE applications SET
    shift = CASE
        WHEN study_type = 'evening' THEN 'evening'
        ELSE 'day'
    END,
    tuition = CASE
        WHEN study_type = 'morning' THEN 'free'
        ELSE 'paid'
    END;

ALTER TABLE applications
    ALTER COLUMN shift SET NOT NULL,
    ADD CONSTRAINT applications_shift_check CHECK (shift IN ('day', 'evening'));

ALTER TABLE applications
    ALTER COLUMN tuition SET NOT NULL,
    ADD CONSTRAINT applications_tuition_check CHECK (tuition IN ('free', 'paid'));

ALTER TABLE applications DROP CONSTRAINT applications_study_type_check;
ALTER TABLE applications DROP COLUMN study_type;

-- Academic rules setting
INSERT INTO settings (id, settings)
SELECT gen_random_uuid(), '{}'::jsonb
WHERE NOT EXISTS (SELECT 1 FROM settings LIMIT 1);

UPDATE settings SET settings = settings || '{"full_year_repeat": false}'::jsonb;
