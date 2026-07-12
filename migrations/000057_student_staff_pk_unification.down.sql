-- Restore the surrogate ids (fresh values — the originals are gone) and
-- repoint the three referrers back through them.

-- ── staff_profiles ──────────────────────────────────────────────────────────

ALTER TABLE staff_profiles DROP CONSTRAINT staff_profiles_pkey;
ALTER TABLE staff_profiles ADD COLUMN id UUID NOT NULL DEFAULT gen_random_uuid();
ALTER TABLE staff_profiles ADD PRIMARY KEY (id);
ALTER TABLE staff_profiles ADD CONSTRAINT staff_profiles_user_id_key UNIQUE (user_id);
CREATE INDEX idx_staff_profiles_user ON staff_profiles(user_id);

-- ── students ────────────────────────────────────────────────────────────────

ALTER TABLE students DROP CONSTRAINT students_pkey;
ALTER TABLE students ADD COLUMN id UUID NOT NULL DEFAULT gen_random_uuid();
ALTER TABLE students ADD PRIMARY KEY (id);
ALTER TABLE students ADD CONSTRAINT students_user_id_key UNIQUE (user_id);

DROP INDEX IF EXISTS idx_students_cursor;
CREATE INDEX idx_students_cursor ON students(created_at DESC, id DESC);

-- ── referrers back onto students(id) ────────────────────────────────────────

UPDATE student_leaves sl
SET student_id = s.id
FROM students s
WHERE s.user_id = sl.student_id;

ALTER TABLE student_leaves DROP CONSTRAINT student_leaves_student_id_fkey;
ALTER TABLE student_leaves
    ADD CONSTRAINT student_leaves_student_id_fkey
    FOREIGN KEY (student_id) REFERENCES students(id) ON DELETE CASCADE;

UPDATE student_cohort_history sch
SET student_id = s.id
FROM students s
WHERE s.user_id = sch.student_id;

ALTER TABLE student_cohort_history DROP CONSTRAINT student_cohort_history_student_id_fkey;
ALTER TABLE student_cohort_history
    ADD CONSTRAINT student_cohort_history_student_id_fkey
    FOREIGN KEY (student_id) REFERENCES students(id) ON DELETE CASCADE;

UPDATE payments p
SET student_id = s.id
FROM students s
WHERE s.user_id = p.student_id;

ALTER TABLE payments DROP CONSTRAINT payments_student_id_fkey;
ALTER TABLE payments
    ADD CONSTRAINT payments_student_id_fkey
    FOREIGN KEY (student_id) REFERENCES students(id) ON DELETE CASCADE;
