-- Fix missing ON DELETE CASCADE constraints that block course deletion.

-- 1. courses.requires (self-ref) → SET NULL so deleting a prerequisite course
--    doesn't block; the dependent course simply loses its prerequisite link.
ALTER TABLE courses
    DROP CONSTRAINT IF EXISTS courses_requires_fkey;
ALTER TABLE courses
    ADD CONSTRAINT courses_requires_fkey
        FOREIGN KEY (requires) REFERENCES courses(id) ON DELETE SET NULL;

-- 2. course_offerings.course_id → CASCADE so offerings are removed with the course.
ALTER TABLE course_offerings
    DROP CONSTRAINT IF EXISTS course_offerings_course_id_fkey;
ALTER TABLE course_offerings
    ADD CONSTRAINT course_offerings_course_id_fkey
        FOREIGN KEY (course_id) REFERENCES courses(id) ON DELETE CASCADE;

-- 3. program_curriculum.course_id → CASCADE so curriculum entries are removed with the course.
ALTER TABLE program_curriculum
    DROP CONSTRAINT IF EXISTS program_curriculum_course_id_fkey;
ALTER TABLE program_curriculum
    ADD CONSTRAINT program_curriculum_course_id_fkey
        FOREIGN KEY (course_id) REFERENCES courses(id) ON DELETE CASCADE;

-- 4. course_enrollments.offering_id → CASCADE so enrollment records are removed with the offering.
ALTER TABLE course_enrollments
    DROP CONSTRAINT IF EXISTS course_enrollments_offering_id_fkey;
ALTER TABLE course_enrollments
    ADD CONSTRAINT course_enrollments_offering_id_fkey
        FOREIGN KEY (offering_id) REFERENCES course_offerings(id) ON DELETE CASCADE;
