DROP INDEX IF EXISTS idx_student_leaves_active;

DROP INDEX IF EXISTS idx_course_offerings_key_live;
ALTER TABLE course_offerings ADD CONSTRAINT course_offerings_unique
    UNIQUE (course_id, semester_id, cohort_year, shift);

DROP INDEX IF EXISTS idx_courses_code_live;
ALTER TABLE courses ADD CONSTRAINT courses_department_id_code_group_order_key
    UNIQUE (department_id, code, group_order);

DROP INDEX IF EXISTS idx_semesters_year_term_live;
ALTER TABLE semesters ADD CONSTRAINT semesters_academic_year_id_semester_key
    UNIQUE (academic_year_id, semester);

ALTER TABLE course_offerings DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE courses DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE semesters DROP COLUMN IF EXISTS deleted_at;

ALTER TABLE course_offerings DROP COLUMN IF EXISTS version;
ALTER TABLE courses DROP COLUMN IF EXISTS version;
ALTER TABLE students DROP COLUMN IF EXISTS version;
