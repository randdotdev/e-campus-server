-- Revert enrollment status constraint
ALTER TABLE course_enrollments DROP CONSTRAINT course_enrollments_status_check;
ALTER TABLE course_enrollments ADD CONSTRAINT course_enrollments_status_check
    CHECK (status IN ('enrolled', 'dropped', 'completed', 'failed'));

-- Drop leave_semesters table
DROP TABLE IF EXISTS leave_semesters;

-- Revert student_leaves changes
ALTER TABLE student_leaves DROP CONSTRAINT IF EXISTS student_leaves_type_check;
ALTER TABLE student_leaves DROP COLUMN IF EXISTS approved_at;
ALTER TABLE student_leaves DROP COLUMN IF EXISTS academic_year_id;
ALTER TABLE student_leaves DROP COLUMN IF EXISTS type;
ALTER TABLE student_leaves ALTER COLUMN start_date SET NOT NULL;
