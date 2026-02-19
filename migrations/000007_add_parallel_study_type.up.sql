-- Add 'parallel' study type for paid morning students
-- morning = free (scholarship/government-funded)
-- evening = paid
-- parallel = paid, but attends morning schedule

ALTER TABLE applications
DROP CONSTRAINT applications_study_type_check;

ALTER TABLE applications
ADD CONSTRAINT applications_study_type_check
CHECK (study_type IN ('morning', 'evening', 'parallel'));
