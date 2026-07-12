-- Management write guards: version CAS tokens, soft deletes with recovery,
-- and the one-active-leave constraint.

-- Optimistic-concurrency tokens (Shape 1) for the remaining mutable
-- aggregates. Applications carry none on purpose: every application write is
-- a status-guarded transition (Shape 2), so a CAS token would guard nothing.
ALTER TABLE students ADD COLUMN IF NOT EXISTS version BIGINT NOT NULL DEFAULT 0;
ALTER TABLE courses ADD COLUMN IF NOT EXISTS version BIGINT NOT NULL DEFAULT 0;
ALTER TABLE course_offerings ADD COLUMN IF NOT EXISTS version BIGINT NOT NULL DEFAULT 0;

-- Soft deletes for the blast-radius entities: hard-deleting a semester,
-- course, or offering cascades into enrollments (student history). Deleted
-- rows are invisible to reads, recoverable by hand, and purged permanently by
-- the janitor after the retention window.
ALTER TABLE semesters ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
ALTER TABLE courses ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
ALTER TABLE course_offerings ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

-- Uniqueness must ignore soft-deleted rows so a deleted entity can be
-- recreated: the table constraints become partial unique indexes over live
-- rows.
ALTER TABLE semesters DROP CONSTRAINT IF EXISTS semesters_academic_year_id_semester_key;
CREATE UNIQUE INDEX IF NOT EXISTS idx_semesters_year_term_live
    ON semesters(academic_year_id, semester) WHERE deleted_at IS NULL;

ALTER TABLE courses DROP CONSTRAINT IF EXISTS courses_department_id_code_group_order_key;
CREATE UNIQUE INDEX IF NOT EXISTS idx_courses_code_live
    ON courses(department_id, code, group_order) WHERE deleted_at IS NULL;

ALTER TABLE course_offerings DROP CONSTRAINT IF EXISTS course_offerings_unique;
CREATE UNIQUE INDEX IF NOT EXISTS idx_course_offerings_key_live
    ON course_offerings(course_id, semester_id, cohort_year, shift) WHERE deleted_at IS NULL;

-- One active (unclosed) leave per student: the database-level guard behind
-- RequestLeave, closing the check-then-act race.
CREATE UNIQUE INDEX IF NOT EXISTS idx_student_leaves_active
    ON student_leaves(student_id) WHERE closed_at IS NULL;
