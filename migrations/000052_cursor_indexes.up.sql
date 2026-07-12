-- Keyset (cursor) pagination indexes.
--
-- Every list endpoint seeks with (sort_col, id) < ($cursor) ORDER BY sort_col
-- DESC, id DESC. That is an index-only seek — O(log n) at any depth — ONLY when
-- an index matches the sort exactly, including the id tiebreak. Migration 000002
-- covered colleges/departments/programs/users; applications and qa_questions got
-- theirs later. The tables below run the same keyset query with no matching
-- composite, so they fall back to a full sort that degrades as they grow. This
-- closes that gap.
--
-- Convention: (sort_col DESC, id DESC), partial on `deleted_at IS NULL` where the
-- list always excludes soft-deleted rows (mirrors idx_qa_questions_cursor).
-- (For very large tables in production, add these CONCURRENTLY out-of-band to
-- avoid a write lock; kept inline here to match the existing migration style.)

-- management: students, courses, offerings, enrollments.
CREATE INDEX IF NOT EXISTS idx_students_cursor
    ON students(created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_courses_cursor
    ON courses(created_at DESC, id DESC) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_course_offerings_cursor
    ON course_offerings(created_at DESC, id DESC) WHERE deleted_at IS NULL;
-- NOTE: enrollments sort on enrolled_at, not created_at.
CREATE INDEX IF NOT EXISTS idx_course_enrollments_cursor
    ON course_enrollments(enrolled_at DESC, id DESC);

-- announcements: the post feed and the activity feed. These had single-column
-- (created_at DESC) indexes that lack the id tiebreak; the composites subsume
-- them, so the old ones are dropped.
DROP INDEX IF EXISTS idx_posts_created;
CREATE INDEX IF NOT EXISTS idx_posts_cursor
    ON posts(created_at DESC, id DESC);
-- A thread's comments seek by root_id, oldest first — its own access path.
CREATE INDEX IF NOT EXISTS idx_posts_thread
    ON posts(root_id, created_at, id) WHERE deleted_at IS NULL;

DROP INDEX IF EXISTS idx_activities_created;
CREATE INDEX IF NOT EXISTS idx_activities_cursor
    ON activities(created_at DESC, id DESC);

-- communication: the notification list is always scoped to one user, so the
-- user_id leads the index. The old (user_id, created_at DESC) lacked the id
-- tiebreak and is subsumed.
DROP INDEX IF EXISTS idx_notifications_user_created;
CREATE INDEX IF NOT EXISTS idx_notifications_user_cursor
    ON notifications(user_id, created_at DESC, id DESC);
