-- students and staff_profiles are 1:1 extensions of users (user_id UNIQUE),
-- so their surrogate ids never distinguished anything — they only forced
-- every referrer to pick one of two keys for the same person, the bug class
-- 000056 started cleaning up in classroom. Repoint the three remaining
-- referrers of students(id), then make user_id the primary key on both.

-- ── Phase 1: repoint referrers of students(id) onto the account id ─────────

UPDATE student_leaves sl
SET student_id = s.user_id
FROM students s
WHERE s.id = sl.student_id;

ALTER TABLE student_leaves DROP CONSTRAINT student_leaves_student_id_fkey;
ALTER TABLE student_leaves
    ADD CONSTRAINT student_leaves_student_id_fkey
    FOREIGN KEY (student_id) REFERENCES users(id) ON DELETE CASCADE;

UPDATE student_cohort_history sch
SET student_id = s.user_id
FROM students s
WHERE s.id = sch.student_id;

ALTER TABLE student_cohort_history DROP CONSTRAINT student_cohort_history_student_id_fkey;
ALTER TABLE student_cohort_history
    ADD CONSTRAINT student_cohort_history_student_id_fkey
    FOREIGN KEY (student_id) REFERENCES users(id) ON DELETE CASCADE;

UPDATE payments p
SET student_id = s.user_id
FROM students s
WHERE s.id = p.student_id;

ALTER TABLE payments DROP CONSTRAINT payments_student_id_fkey;
ALTER TABLE payments
    ADD CONSTRAINT payments_student_id_fkey
    FOREIGN KEY (student_id) REFERENCES users(id) ON DELETE CASCADE;

-- ── Phase 2: students — user_id becomes the primary key ────────────────────

ALTER TABLE students DROP CONSTRAINT students_pkey;
ALTER TABLE students DROP COLUMN id;
ALTER TABLE students ADD PRIMARY KEY (user_id);
-- The PK index makes the old UNIQUE constraint redundant.
ALTER TABLE students DROP CONSTRAINT students_user_id_key;

-- The keyset cursor pairs created_at with the row key, which changed.
DROP INDEX IF EXISTS idx_students_cursor;
CREATE INDEX idx_students_cursor ON students(created_at DESC, user_id DESC);

-- ── Phase 3: staff_profiles — same disease, no referrers ───────────────────

ALTER TABLE staff_profiles DROP CONSTRAINT staff_profiles_pkey;
ALTER TABLE staff_profiles DROP COLUMN id;
ALTER TABLE staff_profiles ADD PRIMARY KEY (user_id);
ALTER TABLE staff_profiles DROP CONSTRAINT staff_profiles_user_id_key;
DROP INDEX IF EXISTS idx_staff_profiles_user;
