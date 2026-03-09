-- Remove pass threshold
ALTER TABLE semesters DROP COLUMN IF EXISTS pass_threshold;

-- Restore original semester type constraint
ALTER TABLE semesters DROP CONSTRAINT IF EXISTS semesters_semester_check;
ALTER TABLE semesters ADD CONSTRAINT semesters_semester_check
    CHECK (semester IN ('fall', 'spring', 'summer'));
