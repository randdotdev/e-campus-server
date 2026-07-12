-- Drop the cursor indexes and restore the single-column indexes they replaced.
DROP INDEX IF EXISTS idx_students_cursor;
DROP INDEX IF EXISTS idx_courses_cursor;
DROP INDEX IF EXISTS idx_course_offerings_cursor;
DROP INDEX IF EXISTS idx_course_enrollments_cursor;

DROP INDEX IF EXISTS idx_posts_cursor;
DROP INDEX IF EXISTS idx_posts_thread;
CREATE INDEX IF NOT EXISTS idx_posts_created ON posts(created_at DESC);

DROP INDEX IF EXISTS idx_activities_cursor;
CREATE INDEX IF NOT EXISTS idx_activities_created ON activities(created_at DESC);

DROP INDEX IF EXISTS idx_notifications_user_cursor;
CREATE INDEX IF NOT EXISTS idx_notifications_user_created ON notifications(user_id, created_at DESC);
