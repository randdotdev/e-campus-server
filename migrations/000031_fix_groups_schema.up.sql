-- Migration: Fix groups schema
-- 1. Drop old project_groups (per-offering theory/practice) - redundant with cohort_groups
-- 2. Rename course_groups → project_groups (for team submissions)
-- 3. Update lesson_schedules to use cohort_group_id only

-- Step 1: Drop foreign key and column from lesson_schedules
ALTER TABLE lesson_schedules DROP CONSTRAINT IF EXISTS lesson_schedules_group_id_fkey;
DROP INDEX IF EXISTS idx_lesson_schedules_group;
ALTER TABLE lesson_schedules DROP COLUMN IF EXISTS group_id;

-- Step 2: Make cohort_group_id required and add unique constraint
ALTER TABLE lesson_schedules ALTER COLUMN cohort_group_id SET NOT NULL;
ALTER TABLE lesson_schedules ADD CONSTRAINT lesson_schedules_lesson_cohort_group_unique UNIQUE (lesson_id, cohort_group_id);

-- Step 3: Drop old project_groups tables (per-offering, now replaced by cohort_groups)
DROP TABLE IF EXISTS project_group_members;
DROP TABLE IF EXISTS project_groups;

-- Step 4: Rename course_groups to project_groups (for team project submissions)
ALTER TABLE course_group_members RENAME TO project_group_members;
ALTER TABLE course_groups RENAME TO project_groups;

-- Step 5: Update indexes
ALTER INDEX IF EXISTS idx_course_groups_project RENAME TO idx_project_groups_project;
ALTER INDEX IF EXISTS idx_course_groups_leader RENAME TO idx_project_groups_leader;
ALTER INDEX IF EXISTS idx_course_group_members_group RENAME TO idx_project_group_members_group;
ALTER INDEX IF EXISTS idx_course_group_members_student RENAME TO idx_project_group_members_student;

-- Step 6: Update foreign key constraint names
ALTER TABLE project_group_members RENAME CONSTRAINT course_group_members_course_group_id_fkey TO project_group_members_project_group_id_fkey;
ALTER TABLE project_group_members RENAME COLUMN course_group_id TO project_group_id;

-- Step 7: Update project_submissions foreign key
ALTER TABLE project_submissions RENAME CONSTRAINT project_submissions_course_group_id_fkey TO project_submissions_project_group_id_fkey;
ALTER TABLE project_submissions RENAME COLUMN course_group_id TO project_group_id;
