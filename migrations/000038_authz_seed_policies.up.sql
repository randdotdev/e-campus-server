-- ── Offering list / get / update ───────────────────────────────────────────
INSERT INTO authz_policies (resource, verb, course_role) VALUES
  ('offering', 'list',   'teacher'),
  ('offering', 'list',   'assistant'),
  ('offering', 'list',   'student'),
  ('offering', 'list',   'observer'),
  ('offering', 'get',    'teacher'),
  ('offering', 'get',    'assistant'),
  ('offering', 'update', 'teacher'),
  ('offering', 'update', 'assistant');

-- offering.update is a teaching-only verb; scope admins use enrollment.update for moderation
INSERT INTO authz_policies (resource, verb, scope_type, min_level) VALUES
  ('offering', 'get',    'department', 'viewer'),
  ('offering', 'get',    'university', 'viewer'),
  ('offering', 'get',    'platform',   'admin');

-- ── Structural resources ───────────────────────────────────────────────────
INSERT INTO authz_policies (resource, verb, scope_type, min_level) VALUES
  ('department', 'get',    'department', 'viewer'),
  ('department', 'get',    'university', 'viewer'),
  ('department', 'get',    'platform',   'admin'),
  ('department', 'update', 'department', 'admin'),
  ('department', 'update', 'university', 'admin'),
  ('department', 'update', 'platform',   'admin'),
  ('college',    'get',    'college',    'viewer'),
  ('college',    'get',    'university', 'viewer'),
  ('college',    'get',    'platform',   'admin'),
  ('college',    'update', 'college',    'admin'),
  ('college',    'update', 'university', 'admin'),
  ('college',    'update', 'platform',   'admin'),
  ('program',    'get',    'program',    'viewer'),
  ('program',    'get',    'department', 'viewer'),
  ('program',    'get',    'university', 'viewer'),
  ('program',    'get',    'platform',   'admin'),
  ('program',    'update', 'program',    'admin'),
  ('program',    'update', 'department', 'admin'),
  ('program',    'update', 'university', 'admin'),
  ('program',    'update', 'platform',   'admin'),
  ('university', 'get',    'university', 'viewer'),
  ('university', 'get',    'platform',   'admin'),
  ('university', 'update', 'university', 'admin'),
  ('university', 'update', 'platform',   'admin'),
  ('platform',   'get',    'platform',   'admin'),
  ('platform',   'update', 'platform',   'admin');

-- ── Academic year / Semester ───────────────────────────────────────────────
INSERT INTO authz_policies (resource, verb, scope_type, min_level) VALUES
  ('academic_year', 'get',    'university', 'viewer'),
  ('academic_year', 'get',    'platform',   'admin'),
  ('academic_year', 'create', 'university', 'admin'),
  ('academic_year', 'create', 'platform',   'admin'),
  ('academic_year', 'update', 'university', 'admin'),
  ('academic_year', 'update', 'platform',   'admin'),
  ('semester', 'get',    'university', 'viewer'),
  ('semester', 'get',    'platform',   'admin'),
  ('semester', 'create', 'university', 'admin'),
  ('semester', 'create', 'platform',   'admin'),
  ('semester', 'update', 'university', 'admin'),
  ('semester', 'update', 'platform',   'admin');


-- ── Enrollment ─────────────────────────────────────────────────────────────
INSERT INTO authz_policies (resource, verb, course_role) VALUES
  ('enrollment', 'create', 'teacher'),
  ('enrollment', 'create', 'assistant'),
  ('enrollment', 'update', 'teacher'),
  ('enrollment', 'update', 'assistant'),
  ('enrollment', 'delete', 'teacher'),
  ('enrollment', 'delete', 'assistant');


-- enrollment.update also gates course-level moderation (muting); dept/college admins need it too
INSERT INTO authz_policies (resource, verb, scope_type, min_level) VALUES
  ('enrollment', 'list',   'university', 'admin'),
  ('enrollment', 'list',   'platform',   'admin'),
  ('enrollment', 'get',    'university', 'admin'),
  ('enrollment', 'get',    'platform',   'admin'),
  ('enrollment', 'update', 'department', 'admin'),
  ('enrollment', 'update', 'college',    'admin'),
  ('enrollment', 'update', 'university', 'admin'),
  ('enrollment', 'update', 'platform',   'admin');
-- ── User ───────────────────────────────────────────────────────────────────
INSERT INTO authz_policies (resource, verb, scope_type, min_level) VALUES
  ('user', 'get',    'university', 'admin'),
  ('user', 'get',    'platform',   'admin'),
  ('user', 'list',   'university', 'admin'),
  ('user', 'list',   'platform',   'admin'),
  ('user', 'create', 'university', 'admin'),
  ('user', 'create', 'platform',   'admin'),
  ('user', 'update', 'university', 'admin'),
  ('user', 'update', 'platform',   'admin'),
  ('user', 'delete', 'university', 'admin'),
  ('user', 'delete', 'platform',   'admin');

-- ── Application ────────────────────────────────────────────────────────────
INSERT INTO authz_policies (resource, verb, scope_type, min_level) VALUES
  ('application', 'list',   'program',    'admin'),
  ('application', 'list',   'department', 'admin'),
  ('application', 'list',   'college',    'admin'),
  ('application', 'list',   'university', 'admin'),
  ('application', 'list',   'platform',   'admin'),
  ('application', 'get',    'program',    'admin'),
  ('application', 'get',    'department', 'admin'),
  ('application', 'get',    'college',    'admin'),
  ('application', 'get',    'university', 'admin'),
  ('application', 'get',    'platform',   'admin'),
  ('application', 'update', 'program',    'admin'),
  ('application', 'update', 'department', 'admin'),
  ('application', 'update', 'college',    'admin'),
  ('application', 'update', 'university', 'admin'),
  ('application', 'update', 'platform',   'admin');

-- ── Subscription ───────────────────────────────────────────────────────────
INSERT INTO authz_policies (resource, verb, scope_type, min_level) VALUES
  ('subscription', 'get',    'university', 'admin'),
  ('subscription', 'get',    'platform',   'admin'),
  ('subscription', 'update', 'platform',   'admin');

-- ── News ───────────────────────────────────────────────────────────────────
INSERT INTO authz_policies (resource, verb, scope_type, min_level) VALUES
  ('news', 'create', 'university', 'admin'),
  ('news', 'create', 'college',    'admin'),
  ('news', 'create', 'department', 'admin');

-- ── Post ───────────────────────────────────────────────────────────────────
INSERT INTO authz_policies (resource, verb, scope_type, min_level) VALUES
  ('post', 'create', 'university', 'admin'),
  ('post', 'create', 'college',    'admin'),
  ('post', 'create', 'department', 'admin'),
  ('post', 'create', 'program',    'admin');

-- ── QA ─────────────────────────────────────────────────────────────────────
INSERT INTO authz_policies (resource, verb, course_role) VALUES
  ('qa', 'create', 'teacher'),
  ('qa', 'create', 'assistant'),
  ('qa', 'create', 'student'),
  ('qa', 'create', 'observer'),
  ('qa', 'update', 'teacher'),
  ('qa', 'update', 'assistant'),
  ('qa', 'update', 'student'),
  ('qa', 'get',    'teacher'),
  ('qa', 'get',    'assistant'),
  ('qa', 'get',    'student'),
  ('qa', 'get',    'observer');

INSERT INTO authz_policies (resource, verb, scope_type, min_level) VALUES
  ('qa', 'update', 'department', 'admin'),
  ('qa', 'update', 'college',    'admin'),
  ('qa', 'update', 'university', 'admin'),
  ('qa', 'update', 'platform',   'admin'),
  ('qa', 'get',    'department', 'viewer'),
  ('qa', 'get',    'university', 'viewer'),
  ('qa', 'get',    'platform',   'admin');

-- qa.delete is author-only at service level (CanDeleteQuestion checks CreatedBy).
-- Only students can delete their own pending questions; teachers reject instead of delete.
INSERT INTO authz_policies (resource, verb, course_role) VALUES
  ('qa', 'delete', 'student');

-- ── Curriculum ─────────────────────────────────────────────────────────────
INSERT INTO authz_policies (resource, verb, scope_type, min_level) VALUES
  ('curriculum', 'update', 'program',    'admin'),
  ('curriculum', 'update', 'department', 'admin'),
  ('curriculum', 'update', 'college',    'admin'),
  ('curriculum', 'update', 'university', 'admin'),
  ('curriculum', 'update', 'platform',   'admin');

-- ── Settings ───────────────────────────────────────────────────────────────
INSERT INTO authz_policies (resource, verb, scope_type, min_level) VALUES
  ('settings', 'update', 'university', 'admin'),
  ('settings', 'update', 'platform',   'admin');

-- ── Policy management ────────────────────────────────────────────────────────
INSERT INTO authz_policies (resource, verb, scope_type, min_level) VALUES
  ('policy', 'create', 'platform',   'super_admin'),
  ('policy', 'create', 'university', 'super_admin'),
  ('policy', 'create', 'college',    'super_admin'),
  ('policy', 'create', 'department', 'super_admin'),
  ('policy', 'create', 'program',    'super_admin'),
  ('policy', 'list',   'platform',   'super_admin'),
  ('policy', 'list',   'university', 'super_admin'),
  ('policy', 'list',   'college',    'super_admin'),
  ('policy', 'list',   'department', 'super_admin'),
  ('policy', 'list',   'program',    'super_admin'),
  ('policy', 'get',    'platform',   'super_admin'),
  ('policy', 'get',    'university', 'super_admin'),
  ('policy', 'get',    'college',    'super_admin'),
  ('policy', 'get',    'department', 'super_admin'),
  ('policy', 'get',    'program',    'super_admin'),
  ('policy', 'update', 'platform',   'super_admin'),
  ('policy', 'update', 'university', 'super_admin'),
  ('policy', 'update', 'college',    'super_admin'),
  ('policy', 'update', 'department', 'super_admin'),
  ('policy', 'update', 'program',    'super_admin'),
  ('policy', 'delete', 'platform',   'super_admin'),
  ('policy', 'delete', 'university', 'super_admin'),
  ('policy', 'delete', 'college',    'super_admin'),
  ('policy', 'delete', 'department', 'super_admin'),
  ('policy', 'delete', 'program',    'super_admin');

-- ── Assignment ──────────────────────────────────────────────────────────────
INSERT INTO authz_policies (resource, verb, course_role) VALUES
  ('assignment', 'create', 'teacher'),
  ('assignment', 'update', 'teacher'),
  ('assignment', 'update', 'assistant'),
  ('assignment', 'get',    'teacher'),
  ('assignment', 'get',    'assistant'),
  ('assignment', 'get',    'student'),
  ('assignment', 'get',    'observer');

INSERT INTO authz_policies (resource, verb, scope_type, min_level) VALUES
  ('assignment', 'get',    'department', 'viewer'),
  ('assignment', 'get',    'university', 'viewer'),
  ('assignment', 'get',    'platform',   'admin');

INSERT INTO authz_policies (resource, verb, course_role) VALUES
  ('assignment', 'delete', 'teacher');

-- ── Exam ────────────────────────────────────────────────────────────────────
INSERT INTO authz_policies (resource, verb, course_role) VALUES
  ('exam', 'create', 'teacher'),
  ('exam', 'get',    'teacher'),
  ('exam', 'get',    'assistant'),
  ('exam', 'get',    'student'),
  ('exam', 'get',    'observer');

INSERT INTO authz_policies (resource, verb, scope_type, min_level) VALUES
  ('exam', 'get',    'department', 'viewer'),
  ('exam', 'get',    'university', 'viewer'),
  ('exam', 'get',    'platform',   'admin');

INSERT INTO authz_policies (resource, verb, course_role) VALUES
  ('exam', 'update', 'teacher'),
  ('exam', 'delete', 'teacher');

-- exam.create scope rows exist solely for question-bank access (scoped by course_code, not offering)
INSERT INTO authz_policies (resource, verb, scope_type, min_level) VALUES
  ('exam', 'create', 'department', 'admin'),
  ('exam', 'create', 'university', 'admin'),
  ('exam', 'create', 'platform',   'admin');

-- ── Grade ───────────────────────────────────────────────────────────────────
INSERT INTO authz_policies (resource, verb, course_role) VALUES
  ('grade', 'create', 'teacher'),
  ('grade', 'get',    'teacher'),
  ('grade', 'get',    'assistant'),
  ('grade', 'get',    'student'),
  ('grade', 'get',    'observer'),
  ('grade', 'update', 'teacher');

-- ── Project ─────────────────────────────────────────────────────────────────
INSERT INTO authz_policies (resource, verb, course_role) VALUES
  ('project', 'create', 'teacher'),
  ('project', 'update', 'teacher'),
  ('project', 'update', 'assistant'),
  ('project', 'delete', 'teacher'),
  ('project', 'get',    'teacher'),
  ('project', 'get',    'assistant'),
  ('project', 'get',    'student'),
  ('project', 'get',    'observer');

INSERT INTO authz_policies (resource, verb, scope_type, min_level) VALUES
  ('project', 'get',    'department', 'viewer'),
  ('project', 'get',    'university', 'viewer'),
  ('project', 'get',    'platform',   'admin');

-- ── Student ─────────────────────────────────────────────────────────────────
INSERT INTO authz_policies (resource, verb, scope_type, min_level) VALUES
  ('student', 'list',   'department', 'viewer'),
  ('student', 'list',   'college',    'viewer'),
  ('student', 'list',   'university', 'viewer'),
  ('student', 'list',   'platform',   'admin'),
  ('student', 'get',    'program',    'viewer'),
  ('student', 'get',    'department', 'viewer'),
  ('student', 'get',    'college',    'viewer'),
  ('student', 'get',    'university', 'viewer'),
  ('student', 'get',    'platform',   'admin'),
  ('student', 'create', 'program',    'admin'),
  ('student', 'create', 'department', 'admin'),
  ('student', 'create', 'university', 'admin'),
  ('student', 'create', 'platform',   'admin'),
  ('student', 'update', 'program',    'admin'),
  ('student', 'update', 'department', 'admin'),
  ('student', 'update', 'university', 'admin'),
  ('student', 'update', 'platform',   'admin'),
  ('student', 'delete', 'university', 'admin'),
  ('student', 'delete', 'platform',   'admin');
-- ── Attendance ──────────────────────────────────────────────────────────────
-- update = teaching task (mark/record); no scope admins
INSERT INTO authz_policies (resource, verb, course_role) VALUES
  ('attendance', 'update', 'teacher'),
  ('attendance', 'update', 'assistant'),
  ('attendance', 'get',    'teacher'),
  ('attendance', 'get',    'assistant');

-- admins can view attendance for oversight (moderation task)
INSERT INTO authz_policies (resource, verb, scope_type, min_level) VALUES
  ('attendance', 'get',    'department', 'viewer'),
  ('attendance', 'get',    'university', 'viewer'),
  ('attendance', 'get',    'platform',   'admin');
