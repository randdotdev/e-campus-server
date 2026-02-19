-- Revert to original study types (will fail if parallel data exists)
ALTER TABLE applications
DROP CONSTRAINT applications_study_type_check;

ALTER TABLE applications
ADD CONSTRAINT applications_study_type_check
CHECK (study_type IN ('morning', 'evening'));
