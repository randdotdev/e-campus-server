-- Revert cascade constraints back to their original definitions.

ALTER TABLE courses
    DROP CONSTRAINT IF EXISTS courses_requires_fkey;
ALTER TABLE courses
    ADD CONSTRAINT courses_requires_fkey
        FOREIGN KEY (requires) REFERENCES courses(id);

ALTER TABLE course_offerings
    DROP CONSTRAINT IF EXISTS course_offerings_course_id_fkey;
ALTER TABLE course_offerings
    ADD CONSTRAINT course_offerings_course_id_fkey
        FOREIGN KEY (course_id) REFERENCES courses(id);

ALTER TABLE program_curriculum
    DROP CONSTRAINT IF EXISTS program_curriculum_course_id_fkey;
ALTER TABLE program_curriculum
    ADD CONSTRAINT program_curriculum_course_id_fkey
        FOREIGN KEY (course_id) REFERENCES courses(id) ON DELETE RESTRICT;

ALTER TABLE course_enrollments
    DROP CONSTRAINT IF EXISTS course_enrollments_offering_id_fkey;
ALTER TABLE course_enrollments
    ADD CONSTRAINT course_enrollments_offering_id_fkey
        FOREIGN KEY (offering_id) REFERENCES course_offerings(id);
