-- Reverse migration: Restore original schema

-- Step 1: Rename columns back
ALTER TABLE project_submissions RENAME COLUMN project_group_id TO course_group_id;
ALTER TABLE project_group_members RENAME COLUMN project_group_id TO course_group_id;

-- Step 2: Rename tables back
ALTER TABLE project_groups RENAME TO course_groups;
ALTER TABLE project_group_members RENAME TO course_group_members;

-- Step 3: Recreate old project_groups tables
CREATE TABLE project_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    offering_id UUID NOT NULL REFERENCES course_offerings(id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL CHECK (type IN ('theory', 'practice')),
    name VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(offering_id, type, name)
);

CREATE TABLE project_group_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    group_id UUID NOT NULL REFERENCES project_groups(id) ON DELETE CASCADE,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(student_id, group_id)
);

CREATE INDEX idx_project_groups_offering ON project_groups(offering_id);
CREATE INDEX idx_project_group_members_student ON project_group_members(student_id);
CREATE INDEX idx_project_group_members_group ON project_group_members(group_id);

-- Step 4: Restore lesson_schedules.group_id
ALTER TABLE lesson_schedules DROP CONSTRAINT IF EXISTS lesson_schedules_lesson_cohort_group_unique;
ALTER TABLE lesson_schedules ALTER COLUMN cohort_group_id DROP NOT NULL;
ALTER TABLE lesson_schedules ADD COLUMN group_id UUID REFERENCES project_groups(id) ON DELETE CASCADE;
CREATE INDEX idx_lesson_schedules_group ON lesson_schedules(group_id);
