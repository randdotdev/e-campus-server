-- Revert domain fix
UPDATE roles SET domain = '' WHERE domain = 'administration';

-- Remove observer role
DELETE FROM course_teachers WHERE role = 'observer';
ALTER TABLE course_teachers DROP CONSTRAINT IF EXISTS course_teachers_role_check;
ALTER TABLE course_teachers ADD CONSTRAINT course_teachers_role_check
    CHECK (role IN ('teacher', 'assistant'));