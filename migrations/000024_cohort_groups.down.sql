-- Down migration: Revert cohort groups changes

-- 1. Remove cohort_group_id from lesson_schedules
DROP INDEX IF EXISTS idx_lesson_schedules_cohort_group;
ALTER TABLE lesson_schedules DROP COLUMN IF EXISTS cohort_group_id;

-- 2. Rename indexes back
ALTER INDEX idx_project_groups_offering RENAME TO idx_groups_offering;
ALTER INDEX idx_project_group_members_student RENAME TO idx_student_groups_student;
ALTER INDEX idx_project_group_members_group RENAME TO idx_student_groups_group;

-- 3. Rename tables back
ALTER TABLE project_group_members RENAME TO student_groups;
ALTER TABLE project_groups RENAME TO groups;

-- 4. Drop cohort group tables
DROP TABLE IF EXISTS student_cohort_groups;
DROP TABLE IF EXISTS cohort_groups;
