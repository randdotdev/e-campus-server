-- Classroom rebuild: version columns for optimistic concurrency, the last
-- uncounted file references become counted inode FKs, and the one-open-
-- attempt rule becomes a partial unique index. Touches classroom tables
-- only.

-- ── Phase 1: version columns (Shape 1 CAS) ────────────────────────────────

ALTER TABLE sections     ADD COLUMN version BIGINT NOT NULL DEFAULT 1;
ALTER TABLE lessons      ADD COLUMN version BIGINT NOT NULL DEFAULT 1;
ALTER TABLE assignments  ADD COLUMN version BIGINT NOT NULL DEFAULT 1;
ALTER TABLE exams        ADD COLUMN version BIGINT NOT NULL DEFAULT 1;
ALTER TABLE qa_questions ADD COLUMN version BIGINT NOT NULL DEFAULT 1;
ALTER TABLE teams        ADD COLUMN version BIGINT NOT NULL DEFAULT 1;
ALTER TABLE projects     ADD COLUMN version BIGINT NOT NULL DEFAULT 1;

-- ── Phase 2: question images become counted references ────────────────────
-- 000055's backfill already counted the UUIDs it harvested from image_url,
-- so converting the column to an FK adds no counts; URLs that never
-- matched an inode are dropped (they pointed at nothing reachable).

ALTER TABLE questions ADD COLUMN image_id UUID REFERENCES inodes(id);

UPDATE questions q
SET image_id = m.iid
FROM (
    SELECT q2.id AS qid, (regexp_matches(q2.image_url,
        '[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}'))[1]::uuid AS iid
    FROM questions q2
    WHERE q2.image_url IS NOT NULL
) m
WHERE q.id = m.qid AND EXISTS (SELECT 1 FROM inodes i WHERE i.id = m.iid);

ALTER TABLE questions DROP COLUMN image_url;
CREATE INDEX idx_questions_image ON questions (image_id) WHERE image_id IS NOT NULL;

-- ── Phase 3: QA attachments join the counting law ─────────────────────────
-- The old rows carried file_path strings — references the counting law
-- cannot see. Paths that resolve to an inode become counted FKs (000055
-- never harvested these columns, so the counts are added here); the rest
-- are dropped — they pointed at storage the new system cannot serve.

ALTER TABLE qa_question_attachments ADD COLUMN stored_file_id UUID REFERENCES inodes(id);
ALTER TABLE qa_answer_attachments   ADD COLUMN stored_file_id UUID REFERENCES inodes(id);

UPDATE qa_question_attachments a
SET stored_file_id = m.iid
FROM (
    SELECT a2.id AS aid, (regexp_matches(a2.file_path,
        '[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}'))[1]::uuid AS iid
    FROM qa_question_attachments a2
) m
WHERE a.id = m.aid AND EXISTS (SELECT 1 FROM inodes i WHERE i.id = m.iid);

UPDATE qa_answer_attachments a
SET stored_file_id = m.iid
FROM (
    SELECT a2.id AS aid, (regexp_matches(a2.file_path,
        '[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}'))[1]::uuid AS iid
    FROM qa_answer_attachments a2
) m
WHERE a.id = m.aid AND EXISTS (SELECT 1 FROM inodes i WHERE i.id = m.iid);

DELETE FROM qa_question_attachments WHERE stored_file_id IS NULL;
DELETE FROM qa_answer_attachments   WHERE stored_file_id IS NULL;

UPDATE inodes i SET link_count = link_count + refs.n
FROM (
    SELECT stored_file_id AS iid, COUNT(*) AS n FROM (
        SELECT stored_file_id FROM qa_question_attachments
        UNION ALL
        SELECT stored_file_id FROM qa_answer_attachments
    ) qa GROUP BY stored_file_id
) refs
WHERE i.id = refs.iid;

ALTER TABLE qa_question_attachments
    ALTER COLUMN stored_file_id SET NOT NULL,
    ADD COLUMN order_index INT NOT NULL DEFAULT 0,
    DROP COLUMN file_path,
    DROP COLUMN file_size,
    DROP COLUMN mime_type;
ALTER TABLE qa_question_attachments RENAME COLUMN file_name TO display_name;

ALTER TABLE qa_answer_attachments
    ALTER COLUMN stored_file_id SET NOT NULL,
    ADD COLUMN order_index INT NOT NULL DEFAULT 0,
    DROP COLUMN file_path,
    DROP COLUMN file_size,
    DROP COLUMN mime_type;
ALTER TABLE qa_answer_attachments RENAME COLUMN file_name TO display_name;

-- ── Phase 4: exam attempts key on the account ──────────────────────────────
-- The whole context keys people by users.id; exam_attempts was the one
-- holdout on students.id. Repoint through the students row, then swap the
-- FK. (students.user_id is UNIQUE, so the mapping is 1:1.)

UPDATE exam_attempts ea
SET student_id = st.user_id
FROM students st
WHERE st.id = ea.student_id;

ALTER TABLE exam_attempts DROP CONSTRAINT exam_attempts_student_id_fkey;
ALTER TABLE exam_attempts
    ADD CONSTRAINT exam_attempts_student_id_fkey
    FOREIGN KEY (student_id) REFERENCES users(id) ON DELETE CASCADE;

-- ── Phase 5: teams bind to a program and cohort ─────────────────────────────
-- Classmates group with classmates: a team carries its creator's program
-- and cohort, and membership is guarded against them. Teams whose leader
-- has no student record cannot satisfy the rule and are dropped (members
-- ride the FK cascade).

DELETE FROM teams t
WHERE NOT EXISTS (SELECT 1 FROM students st WHERE st.user_id = t.leader_id);

ALTER TABLE teams
    ADD COLUMN program_id UUID REFERENCES programs(id) ON DELETE CASCADE,
    ADD COLUMN cohort_year INT;

UPDATE teams t
SET program_id = st.program_id, cohort_year = st.current_cohort_year
FROM students st
WHERE st.user_id = t.leader_id;

ALTER TABLE teams
    ALTER COLUMN program_id SET NOT NULL,
    ALTER COLUMN cohort_year SET NOT NULL;
CREATE INDEX idx_teams_program_cohort ON teams (program_id, cohort_year);

-- ── Phase 6: show_results collapses to three states ─────────────────────────
-- 'immediately' and 'after_submit' were never distinguishable in behaviour;
-- 'after_deadline' becomes 'after_close' (the window closing is the event
-- that exists).

UPDATE exams SET show_results = 'after_submit' WHERE show_results = 'immediately';
UPDATE exams SET show_results = 'after_close'  WHERE show_results = 'after_deadline';
ALTER TABLE exams DROP CONSTRAINT exams_show_results_check;
ALTER TABLE exams ADD CONSTRAINT exams_show_results_check
    CHECK (show_results IN ('after_submit', 'after_close', 'manual'));

-- ── Phase 7: one open attempt per (exam, student) ──────────────────────────
-- Duplicate open attempts (the old check-then-act race's droppings) lose
-- all but the most recently started; then the index makes the rule law,
-- and StartAttempt's ON CONFLICT infers it.

DELETE FROM exam_attempts a
USING exam_attempts b
WHERE a.exam_id = b.exam_id AND a.student_id = b.student_id
  AND a.submitted_at IS NULL AND b.submitted_at IS NULL
  AND (a.started_at < b.started_at OR (a.started_at = b.started_at AND a.id < b.id));

CREATE UNIQUE INDEX idx_exam_attempts_one_open
    ON exam_attempts (exam_id, student_id) WHERE submitted_at IS NULL;
