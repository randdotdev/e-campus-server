-- Rename ects to credits
ALTER TABLE courses RENAME COLUMN ects TO credits;
ALTER TABLE programs RENAME COLUMN total_ects TO total_credits;
ALTER TABLE semester_requirements RENAME COLUMN min_ects TO min_credits;

-- Rename year to stage
ALTER TABLE program_curriculum RENAME COLUMN year TO stage;
ALTER TABLE semester_requirements RENAME COLUMN year TO stage;

-- Simplify semester_requirements (remove effective dates, use audit_logs instead)
ALTER TABLE semester_requirements DROP CONSTRAINT IF EXISTS semester_requirements_no_overlap;
ALTER TABLE semester_requirements DROP COLUMN IF EXISTS effective_from;
ALTER TABLE semester_requirements DROP COLUMN IF EXISTS effective_to;
ALTER TABLE semester_requirements ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT NOW();

-- Add unique constraint (simpler than overlap exclusion)
ALTER TABLE semester_requirements
    ADD CONSTRAINT semester_requirements_unique
    UNIQUE (program_id, cohort_year, stage, semester);

-- Add trigger for updated_at
CREATE TRIGGER update_semester_requirements_updated_at
    BEFORE UPDATE ON semester_requirements
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
