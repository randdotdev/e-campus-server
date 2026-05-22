ALTER TABLE course_offerings
    DROP CONSTRAINT IF EXISTS course_offerings_unique;

ALTER TABLE course_offerings
    ADD CONSTRAINT course_offerings_course_id_semester_id_shift_key
    UNIQUE (course_id, semester_id, shift);
