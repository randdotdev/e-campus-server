-- Revert applications: shift + tuition -> study_type
ALTER TABLE applications ADD COLUMN study_type VARCHAR(20);

UPDATE applications SET
    study_type = CASE
        WHEN shift = 'evening' THEN 'evening'
        WHEN tuition = 'free' THEN 'morning'
        ELSE 'parallel'
    END;

ALTER TABLE applications
    ALTER COLUMN study_type SET NOT NULL,
    ADD CONSTRAINT applications_study_type_check
        CHECK (study_type IN ('morning', 'parallel', 'evening'));

ALTER TABLE applications DROP CONSTRAINT applications_tuition_check;
ALTER TABLE applications DROP COLUMN tuition;

ALTER TABLE applications DROP CONSTRAINT applications_shift_check;
ALTER TABLE applications DROP COLUMN shift;

-- Drop tables
DROP TABLE IF EXISTS payments;
DROP TABLE IF EXISTS semester_requirements;
DROP TABLE IF EXISTS program_curriculum;
DROP TABLE IF EXISTS student_cohort_history;
DROP TABLE IF EXISTS student_leaves;
DROP TABLE IF EXISTS students;

-- Remove setting
UPDATE settings SET settings = settings - 'full_year_repeat';
