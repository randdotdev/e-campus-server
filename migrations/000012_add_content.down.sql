DROP TABLE IF EXISTS lesson_schedules;
DROP TABLE IF EXISTS lesson_attachments;

ALTER TABLE lessons DROP COLUMN IF EXISTS body;
ALTER TABLE lessons DROP COLUMN IF EXISTS mode;
ALTER TABLE lessons DROP COLUMN IF EXISTS unlock_at;
ALTER TABLE lessons DROP COLUMN IF EXISTS attendance_required;
ALTER TABLE lessons DROP COLUMN IF EXISTS allow_download;

ALTER TABLE lessons DROP CONSTRAINT IF EXISTS lessons_type_check;
ALTER TABLE lessons ADD COLUMN offering_id UUID REFERENCES course_offerings(id);
ALTER TABLE lessons ADD COLUMN description TEXT;
ALTER TABLE lessons ADD COLUMN scheduled_at TIMESTAMPTZ;
ALTER TABLE lessons ADD COLUMN room VARCHAR(50);
ALTER TABLE lessons ADD COLUMN publish_at TIMESTAMPTZ;
ALTER TABLE lessons ALTER COLUMN type SET NOT NULL;
ALTER TABLE lessons ADD CONSTRAINT lessons_type_check CHECK (type IN ('theory', 'practice', 'other'));

CREATE INDEX idx_lessons_offering ON lessons(offering_id);
CREATE INDEX idx_lessons_scheduled ON lessons(scheduled_at);

DROP TABLE IF EXISTS student_groups;
DROP TABLE IF EXISTS groups;
