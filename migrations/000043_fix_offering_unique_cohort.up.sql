-- Fix course_offerings unique constraint to include cohort_year.
-- Previously (course_id, semester_id, shift) allowed only one offering per
-- course per semester per shift regardless of cohort, meaning students from
-- different cohort years would incorrectly share a single offering record.
-- The correct key is (course_id, semester_id, cohort_year, shift).

ALTER TABLE course_offerings
    DROP CONSTRAINT IF EXISTS course_offerings_course_id_semester_id_shift_key;

ALTER TABLE course_offerings
    ADD CONSTRAINT course_offerings_unique
    UNIQUE (course_id, semester_id, cohort_year, shift);
