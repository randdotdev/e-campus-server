-- Add pass threshold to semesters (default 50%)
ALTER TABLE semesters ADD COLUMN IF NOT EXISTS pass_threshold INT NOT NULL DEFAULT 50;

-- Add 'annual' semester type for institutions with yearly semesters
ALTER TABLE semesters DROP CONSTRAINT IF EXISTS semesters_semester_check;
ALTER TABLE semesters ADD CONSTRAINT semesters_semester_check
    CHECK (semester IN ('fall', 'spring', 'summer', 'annual'));
