-- Reverse of the classroom rebuild. The QA file_path strings and question
-- image URLs cannot be resurrected from FKs verbatim; the down migration
-- restores the columns with the canonical object path for each inode.

DROP INDEX IF EXISTS idx_exam_attempts_one_open;

-- show_results back to the four-value set.
ALTER TABLE exams DROP CONSTRAINT exams_show_results_check;
ALTER TABLE exams ADD CONSTRAINT exams_show_results_check
    CHECK (show_results IN ('immediately', 'after_submit', 'after_deadline', 'manual'));
UPDATE exams SET show_results = 'after_deadline' WHERE show_results = 'after_close';

-- teams unbind.
DROP INDEX IF EXISTS idx_teams_program_cohort;
ALTER TABLE teams DROP COLUMN program_id, DROP COLUMN cohort_year;

-- exam attempts back to the student record.
UPDATE exam_attempts ea
SET student_id = st.id
FROM students st
WHERE st.user_id = ea.student_id;
ALTER TABLE exam_attempts DROP CONSTRAINT exam_attempts_student_id_fkey;
ALTER TABLE exam_attempts
    ADD CONSTRAINT exam_attempts_student_id_fkey
    FOREIGN KEY (student_id) REFERENCES students(id) ON DELETE CASCADE;

-- QA attachments back to path columns.
ALTER TABLE qa_question_attachments RENAME COLUMN display_name TO file_name;
ALTER TABLE qa_answer_attachments   RENAME COLUMN display_name TO file_name;

ALTER TABLE qa_question_attachments
    ADD COLUMN file_path VARCHAR(500),
    ADD COLUMN file_size INTEGER,
    ADD COLUMN mime_type VARCHAR(100),
    DROP COLUMN order_index;
ALTER TABLE qa_answer_attachments
    ADD COLUMN file_path VARCHAR(500),
    ADD COLUMN file_size INTEGER,
    ADD COLUMN mime_type VARCHAR(100),
    DROP COLUMN order_index;

UPDATE qa_question_attachments a
SET file_path = COALESCE(i.legacy_key, 'sha256/' || i.content_hash),
    file_size = i.size_bytes, mime_type = i.mime_type
FROM inodes i WHERE i.id = a.stored_file_id;
UPDATE qa_answer_attachments a
SET file_path = COALESCE(i.legacy_key, 'sha256/' || i.content_hash),
    file_size = i.size_bytes, mime_type = i.mime_type
FROM inodes i WHERE i.id = a.stored_file_id;

ALTER TABLE qa_question_attachments
    ALTER COLUMN file_path SET NOT NULL,
    ALTER COLUMN file_size SET NOT NULL,
    ALTER COLUMN mime_type SET NOT NULL;
ALTER TABLE qa_answer_attachments
    ALTER COLUMN file_path SET NOT NULL,
    ALTER COLUMN file_size SET NOT NULL,
    ALTER COLUMN mime_type SET NOT NULL;

UPDATE inodes i SET link_count = GREATEST(link_count - refs.n, 0)
FROM (
    SELECT stored_file_id AS iid, COUNT(*) AS n FROM (
        SELECT stored_file_id FROM qa_question_attachments
        UNION ALL
        SELECT stored_file_id FROM qa_answer_attachments
    ) qa GROUP BY stored_file_id
) refs
WHERE i.id = refs.iid;

ALTER TABLE qa_question_attachments DROP COLUMN stored_file_id;
ALTER TABLE qa_answer_attachments   DROP COLUMN stored_file_id;

-- Question images back to URL strings.
DROP INDEX IF EXISTS idx_questions_image;
ALTER TABLE questions ADD COLUMN image_url VARCHAR(500);
UPDATE questions q
SET image_url = '/api/v1/files/' || q.image_id
WHERE q.image_id IS NOT NULL;
ALTER TABLE questions DROP COLUMN image_id;

ALTER TABLE sections     DROP COLUMN version;
ALTER TABLE lessons      DROP COLUMN version;
ALTER TABLE assignments  DROP COLUMN version;
ALTER TABLE exams        DROP COLUMN version;
ALTER TABLE qa_questions DROP COLUMN version;
ALTER TABLE teams        DROP COLUMN version;
ALTER TABLE projects     DROP COLUMN version;
