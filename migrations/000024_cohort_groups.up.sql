-- Migration: Add cohort groups and rename existing groups to project groups
--
-- Cohort groups: per program + cohort_year + stage (for lesson scheduling)
-- Project groups: per offering (for team assignments)

-- 1. Add cohort_groups table (per program + cohort_year + stage)
CREATE TABLE cohort_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_id UUID NOT NULL REFERENCES programs(id) ON DELETE CASCADE,
    cohort_year INT NOT NULL,
    stage INT NOT NULL,
    type VARCHAR(20) NOT NULL CHECK (type IN ('theory', 'practice')),
    name VARCHAR(10) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(program_id, cohort_year, stage, type, name)
);

CREATE INDEX idx_cohort_groups_program ON cohort_groups(program_id);
CREATE INDEX idx_cohort_groups_lookup ON cohort_groups(program_id, cohort_year, stage);

-- 2. Add student_cohort_groups table
CREATE TABLE student_cohort_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    cohort_group_id UUID NOT NULL REFERENCES cohort_groups(id) ON DELETE CASCADE,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(student_id, cohort_group_id)
);

CREATE INDEX idx_student_cohort_groups_student ON student_cohort_groups(student_id);
CREATE INDEX idx_student_cohort_groups_group ON student_cohort_groups(cohort_group_id);

-- 3. Rename groups to project_groups (per-offering team assignments)
ALTER TABLE groups RENAME TO project_groups;

-- 4. Rename student_groups to project_group_members
ALTER TABLE student_groups RENAME TO project_group_members;

-- 5. Update index names for clarity
ALTER INDEX idx_groups_offering RENAME TO idx_project_groups_offering;
ALTER INDEX idx_student_groups_student RENAME TO idx_project_group_members_student;
ALTER INDEX idx_student_groups_group RENAME TO idx_project_group_members_group;

-- 6. Add cohort_group_id to lesson_schedules for new scheduling model
-- Keep group_id temporarily for backward compatibility
ALTER TABLE lesson_schedules ADD COLUMN cohort_group_id UUID REFERENCES cohort_groups(id) ON DELETE CASCADE;
CREATE INDEX idx_lesson_schedules_cohort_group ON lesson_schedules(cohort_group_id);
