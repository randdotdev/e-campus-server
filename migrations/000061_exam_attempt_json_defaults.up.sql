-- exam_attempts.answers/scores were nullable JSONB with no default, but the
-- domain scans them into json.RawMessage, which cannot hold NULL. StartAttempt
-- (exam_id/student_id/started_at only) and RecordResults both leave them NULL,
-- so the RETURNING/SELECT scan 500s on the first student :start. Default them
-- to an empty object and make them NOT NULL so every insert path is scannable.
UPDATE exam_attempts SET answers = '{}'::jsonb WHERE answers IS NULL;
UPDATE exam_attempts SET scores = '{}'::jsonb WHERE scores IS NULL;

ALTER TABLE exam_attempts
    ALTER COLUMN answers SET DEFAULT '{}'::jsonb,
    ALTER COLUMN answers SET NOT NULL,
    ALTER COLUMN scores SET DEFAULT '{}'::jsonb,
    ALTER COLUMN scores SET NOT NULL;
