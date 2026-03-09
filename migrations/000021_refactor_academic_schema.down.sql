-- Remove trigger
DROP TRIGGER IF EXISTS update_semester_requirements_updated_at ON semester_requirements;

-- Remove unique constraint
ALTER TABLE semester_requirements DROP CONSTRAINT IF EXISTS semester_requirements_unique;

-- Restore effective dates
ALTER TABLE semester_requirements DROP COLUMN IF EXISTS updated_at;
ALTER TABLE semester_requirements ADD COLUMN effective_from DATE NOT NULL DEFAULT CURRENT_DATE;
ALTER TABLE semester_requirements ADD COLUMN effective_to DATE;

-- Restore overlap constraint
ALTER TABLE semester_requirements
    ADD CONSTRAINT semester_requirements_no_overlap
    EXCLUDE USING gist (
        program_id WITH =,
        cohort_year WITH =,
        stage WITH =,
        (semester::text) WITH =,
        daterange(effective_from, COALESCE(effective_to, '9999-12-31'::date), '[]') WITH &&
    );

-- Rename stage back to year
ALTER TABLE semester_requirements RENAME COLUMN stage TO year;
ALTER TABLE program_curriculum RENAME COLUMN stage TO year;

-- Rename credits back to ects
ALTER TABLE semester_requirements RENAME COLUMN min_credits TO min_ects;
ALTER TABLE programs RENAME COLUMN total_credits TO total_ects;
ALTER TABLE courses RENAME COLUMN credits TO ects;
