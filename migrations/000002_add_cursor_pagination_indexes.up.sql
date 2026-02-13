-- Indexes for cursor-based pagination
-- These composite indexes optimize queries with ORDER BY created_at DESC, id DESC

CREATE INDEX idx_colleges_cursor ON colleges(created_at DESC, id DESC);
CREATE INDEX idx_departments_cursor ON departments(created_at DESC, id DESC);
CREATE INDEX idx_programs_cursor ON programs(created_at DESC, id DESC);
CREATE INDEX idx_users_cursor ON users(created_at DESC, id DESC);
