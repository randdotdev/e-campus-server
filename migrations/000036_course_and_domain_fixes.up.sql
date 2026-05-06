-- Add observer role to course_teachers
ALTER TABLE course_teachers DROP CONSTRAINT IF EXISTS course_teachers_role_check;
ALTER TABLE course_teachers ADD CONSTRAINT course_teachers_role_check
    CHECK (role IN ('teacher', 'assistant', 'observer'));

-- Fix domain administration
UPDATE roles SET domain = 'administration' WHERE domain = '';