-- Add leave type and academic year reference
ALTER TABLE student_leaves ADD COLUMN type VARCHAR(20) NOT NULL DEFAULT 'short';
ALTER TABLE student_leaves ADD COLUMN academic_year_id UUID REFERENCES academic_years(id);
ALTER TABLE student_leaves ADD COLUMN approved_at TIMESTAMPTZ;

-- Add check constraint for type
ALTER TABLE student_leaves ADD CONSTRAINT student_leaves_type_check
    CHECK (type IN ('short', 'semester', 'year'));

-- Make dates nullable (not needed for academic leaves)
ALTER TABLE student_leaves ALTER COLUMN start_date DROP NOT NULL;

-- Table for semester-based leaves (for semester and year type leaves)
CREATE TABLE leave_semesters (
    leave_id UUID NOT NULL REFERENCES student_leaves(id) ON DELETE CASCADE,
    semester_id UUID NOT NULL REFERENCES semesters(id),
    PRIMARY KEY (leave_id, semester_id)
);

CREATE INDEX idx_leave_semesters_semester ON leave_semesters(semester_id);

-- Add withdrawn_leave status to enrollments
ALTER TABLE course_enrollments DROP CONSTRAINT course_enrollments_status_check;
ALTER TABLE course_enrollments ADD CONSTRAINT course_enrollments_status_check
    CHECK (status IN ('enrolled', 'dropped', 'completed', 'failed', 'withdrawn_leave'));
