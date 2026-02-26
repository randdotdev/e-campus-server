-- Groups for scheduling (theory/practice subdivisions per offering)
CREATE TABLE groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    offering_id UUID NOT NULL REFERENCES course_offerings(id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL CHECK (type IN ('theory', 'practice')),
    name VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(offering_id, type, name)
);

CREATE TABLE student_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    group_id UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(student_id, group_id)
);

CREATE INDEX idx_groups_offering ON groups(offering_id);
CREATE INDEX idx_student_groups_student ON student_groups(student_id);
CREATE INDEX idx_student_groups_group ON student_groups(group_id);

-- Alter lessons table: add new columns, modify existing
ALTER TABLE lessons DROP COLUMN IF EXISTS offering_id;
ALTER TABLE lessons DROP COLUMN IF EXISTS description;
ALTER TABLE lessons DROP COLUMN IF EXISTS scheduled_at;
ALTER TABLE lessons DROP COLUMN IF EXISTS room;
ALTER TABLE lessons DROP COLUMN IF EXISTS publish_at;

ALTER TABLE lessons ADD COLUMN body TEXT;
ALTER TABLE lessons ADD COLUMN mode VARCHAR(20) NOT NULL DEFAULT 'async' CHECK (mode IN ('in_class', 'live', 'async'));
ALTER TABLE lessons ADD COLUMN unlock_at TIMESTAMPTZ;
ALTER TABLE lessons ADD COLUMN attendance_required BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE lessons ADD COLUMN allow_download BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE lessons ALTER COLUMN type DROP NOT NULL;
ALTER TABLE lessons DROP CONSTRAINT IF EXISTS lessons_type_check;
ALTER TABLE lessons ADD CONSTRAINT lessons_type_check CHECK (type IN ('theory', 'practice'));

DROP INDEX IF EXISTS idx_lessons_offering;
DROP INDEX IF EXISTS idx_lessons_scheduled;

-- Lesson attachments (maps display_name to stored_file_id)
CREATE TABLE lesson_attachments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lesson_id UUID NOT NULL REFERENCES lessons(id) ON DELETE CASCADE,
    stored_file_id UUID NOT NULL REFERENCES stored_files(id),
    display_name VARCHAR(255) NOT NULL,
    order_index INT NOT NULL DEFAULT 0,
    added_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(lesson_id, display_name)
);

-- Lesson schedules (per group)
CREATE TABLE lesson_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lesson_id UUID NOT NULL REFERENCES lessons(id) ON DELETE CASCADE,
    group_id UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    scheduled_at TIMESTAMPTZ NOT NULL,
    room VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(lesson_id, group_id)
);

CREATE INDEX idx_lesson_attachments_lesson ON lesson_attachments(lesson_id);
CREATE INDEX idx_lesson_attachments_stored_file ON lesson_attachments(stored_file_id);
CREATE INDEX idx_lesson_schedules_lesson ON lesson_schedules(lesson_id);
CREATE INDEX idx_lesson_schedules_group ON lesson_schedules(group_id);
CREATE INDEX idx_lesson_schedules_time ON lesson_schedules(scheduled_at);
