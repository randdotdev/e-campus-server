-- ═══════════════════════════════════════════════════════════════════════════════
--  AuthZ Policy Seed Data — 246 policies
-- ═══════════════════════════════════════════════════════════════════════════════
--
--  Columns used:
--    resource    → what is being accessed       (e.g. 'course', 'user')
--    verb        → what action is requested     (e.g. 'get', 'create')
--    course_role → granted to anyone with this role in the course context
--    scope_type  → granted to institution roles scoped at/below this level
--    min_level   → minimum role level required when scope_type is set
--
--  Rules:
--  • course_role and scope/min_level are mutually exclusive on a single row.
--  • Scope coverage is hierarchical: platform ≥ university ≥ college ≥
--    department ≥ program.  A university admin implicitly covers all colleges,
--    departments, and programs inside that university.
--  • 'viewer' can read; 'admin' can mutate; 'super_admin' is required for
--    policy management (the most sensitive resource).
--
-- ═══════════════════════════════════════════════════════════════════════════════


-- ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
--  1. STRUCTURAL RESOURCES
-- ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

-- ── Department ─────────────────────────────────────────────────────────────────
--  Viewer access inside the department chain + platform oversight.
INSERT INTO authz_policies (resource, verb, scope_type, min_level) VALUES
  ('department', 'get',    'department', 'viewer'),
  ('department', 'get',    'university', 'viewer'),
  ('department', 'get',    'platform',   'admin'),
  ('department', 'update', 'department', 'admin'),
  ('department', 'update', 'university', 'admin'),
  ('department', 'update', 'platform',   'admin');

-- ── College ────────────────────────────────────────────────────────────────────
INSERT INTO authz_policies (resource, verb, scope_type, min_level) VALUES
  ('college',    'get',    'college',    'viewer'),
  ('college',    'get',    'university', 'viewer'),
  ('college',    'get',    'platform',   'admin'),
  ('college',    'update', 'college',    'admin'),
  ('college',    'update', 'university', 'admin'),
  ('college',    'update', 'platform',   'admin');

-- ── Program ────────────────────────────────────────────────────────────────────
INSERT INTO authz_policies (resource, verb, scope_type, min_level) VALUES
  ('program',    'get',    'program',    'viewer'),
  ('program',    'get',    'department', 'viewer'),
  ('program',    'get',    'university', 'viewer'),
  ('program',    'get',    'platform',   'admin'),
  ('program',    'update', 'program',    'admin'),
  ('program',    'update', 'department', 'admin'),
  ('program',    'update', 'university', 'admin'),
  ('program',    'update', 'platform',   'admin');

-- ── Course ─────────────────────────────────────────────────────────────────────
--  Viewers can read courses within their scope.
--  Create / delete are admin-only and restricted to department+ (courses are
--  curriculum-level objects, not program-level).
INSERT INTO authz_policies (resource, verb, scope_type, min_level) VALUES
  ('course',     'get',    'program',    'viewer'),
  ('course',     'get',    'department', 'viewer'),
  ('course',     'get',    'university', 'viewer'),
  ('course',     'get',    'platform',   'admin'),
  ('course',     'create', 'department', 'admin'),
  ('course',     'create', 'university', 'admin'),
  ('course',     'create', 'platform',   'admin'),
  ('course',     'delete', 'department', 'admin'),
  ('course',     'delete', 'university', 'admin'),
  ('course',     'delete', 'platform',   'admin');

-- ── University ─────────────────────────────────────────────────────────────────
INSERT INTO authz_policies (resource, verb, scope_type, min_level) VALUES
  ('university', 'get',    'university', 'viewer'),
  ('university', 'get',    'platform',   'admin'),
  ('university', 'update', 'university', 'admin'),
  ('university', 'update', 'platform',   'admin');

-- ── Platform ───────────────────────────────────────────────────────────────────
--  Platform is the root scope; only platform admins can touch it.
INSERT INTO authz_policies (resource, verb, scope_type, min_level) VALUES
  ('platform',   'get',    'platform',   'admin'),
  ('platform',   'update', 'platform',   'admin');


-- ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
--  2. ACADEMIC CALENDAR
-- ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

-- ── Academic Year ──────────────────────────────────────────────────────────────
INSERT INTO authz_policies (resource, verb, scope_type, min_level) VALUES
  ('academic_year', 'get',    'university', 'viewer'),
  ('academic_year', 'get',    'platform',   'admin'),
  ('academic_year', 'create', 'university', 'admin'),
  ('academic_year', 'create', 'platform',   'admin'),
  ('academic_year', 'update', 'university', 'admin'),
  ('academic_year', 'update', 'platform',   'admin');

-- ── Semester ───────────────────────────────────────────────────────────────────
INSERT INTO authz_policies (resource, verb, scope_type, min_level) VALUES
  ('semester', 'get',    'university', 'viewer'),
  ('semester', 'get',    'platform',   'admin'),
  ('semester', 'create', 'university', 'admin'),
  ('semester', 'create', 'platform',   'admin'),
  ('semester', 'update', 'university', 'admin'),
  ('semester', 'update', 'platform',   'admin');


-- ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
--  3. COURSE LIFECYCLE
-- ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

-- ── Offering ───────────────────────────────────────────────────────────────────
--  Anyone with a course role (teacher, assistant, student, observer) can list
--  and view offerings they are enrolled in.
INSERT INTO authz_policies (resource, verb, course_role) VALUES
  ('offering', 'list',   'teacher'),
  ('offering', 'list',   'assistant'),
  ('offering', 'list',   'student'),
  ('offering', 'list',   'observer'),
  ('offering', 'get',    'teacher'),
  ('offering', 'get',    'assistant'),
  ('offering', 'get',    'student'),
  ('offering', 'get',    'observer');

--  Course-role access: teaching staff can update their own offering
--  (this gates section, lesson, attachment, schedule, and teacher mutations).
INSERT INTO authz_policies (resource, verb, course_role) VALUES
  ('offering', 'update', 'teacher'),
  ('offering', 'update', 'assistant');

--  Scope-based access:
--  • get    → viewers inside the department chain can read offerings.
--  • list   → admins use this for oversight dashboards.
--  • update → scope admins update for moderation (enrollment.update is the
--             separate gate for enrollment-level moderation).
INSERT INTO authz_policies (resource, verb, scope_type, min_level) VALUES
  ('offering', 'get',    'department', 'viewer'),
  ('offering', 'get',    'university', 'viewer'),
  ('offering', 'get',    'platform',   'admin'),
  ('offering', 'list',   'department', 'admin'),
  ('offering', 'list',   'college',    'admin'),
  ('offering', 'list',   'university', 'admin'),
  ('offering', 'list',   'platform',   'admin'),
  ('offering', 'update', 'department', 'admin'),
  ('offering', 'update', 'college',    'admin'),
  ('offering', 'update', 'university', 'admin'),
  ('offering', 'update', 'platform',   'admin');

-- ── Enrollment ─────────────────────────────────────────────────────────────────
--  Teaching staff manage enrollments for their offerings.
INSERT INTO authz_policies (resource, verb, course_role) VALUES
  ('enrollment', 'create', 'teacher'),
  ('enrollment', 'create', 'assistant'),
  ('enrollment', 'update', 'teacher'),
  ('enrollment', 'update', 'assistant'),
  ('enrollment', 'delete', 'teacher'),
  ('enrollment', 'delete', 'assistant');

--  Scope admins can list / get for reporting; department+ admins can update
--  for moderation (e.g. muting a disruptive student).
INSERT INTO authz_policies (resource, verb, scope_type, min_level) VALUES
  ('enrollment', 'list',   'university', 'admin'),
  ('enrollment', 'list',   'platform',   'admin'),
  ('enrollment', 'get',    'university', 'admin'),
  ('enrollment', 'get',    'platform',   'admin'),
  ('enrollment', 'update', 'department', 'admin'),
  ('enrollment', 'update', 'college',    'admin'),
  ('enrollment', 'update', 'university', 'admin'),
  ('enrollment', 'update', 'platform',   'admin');


-- ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
--  4. USER MANAGEMENT
-- ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

-- ── User ───────────────────────────────────────────────────────────────────────
--  User CRUD is admin-only.  No viewer access — user directory is sensitive.
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

-- ── Student ────────────────────────────────────────────────────────────────────
--  Viewers can read student records within their scope (program → platform).
--  Admins can create / update within their scope; delete is restricted to
--  university+ to prevent accidental data loss at lower scopes.
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

-- ── Application ────────────────────────────────────────────────────────────────
--  Admissions pipeline — all verbs are admin-only, scoped program → platform.
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


-- ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
--  5. COMMUNICATION
-- ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

-- ── Post ───────────────────────────────────────────────────────────────────────
--  General posts (university / college / department / program scope) are
--  created by admins.  Course-scoped posts are gated by course membership
--  in the handler, not by these rows.
INSERT INTO authz_policies (resource, verb, scope_type, min_level) VALUES
  ('post', 'create', 'university', 'admin'),
  ('post', 'create', 'college',    'admin'),
  ('post', 'create', 'department', 'admin'),
  ('post', 'create', 'program',    'admin');

-- ── News ───────────────────────────────────────────────────────────────────────
--  News bulletins are created by admins at university / college / department.
INSERT INTO authz_policies (resource, verb, scope_type, min_level) VALUES
  ('news', 'create', 'university', 'admin'),
  ('news', 'create', 'college',    'admin'),
  ('news', 'create', 'department', 'admin');

-- ── QA (Questions & Answers) ───────────────────────────────────────────────────
--  Course-scoped: all course members can create / get; teachers & assistants
--  can update (moderate / answer); students can delete their own questions.
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
  ('qa', 'get',    'observer'),
  ('qa', 'delete', 'student');

--  Scope-based: viewers can read for oversight; admins can moderate.
INSERT INTO authz_policies (resource, verb, scope_type, min_level) VALUES
  ('qa', 'update', 'department', 'admin'),
  ('qa', 'update', 'college',    'admin'),
  ('qa', 'update', 'university', 'admin'),
  ('qa', 'update', 'platform',   'admin'),
  ('qa', 'get',    'department', 'viewer'),
  ('qa', 'get',    'university', 'viewer'),
  ('qa', 'get',    'platform',   'admin');


-- ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
--  6. COURSE CONTENT & ACADEMIC WORK
-- ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

-- ── Assignment ─────────────────────────────────────────────────────────────────
--  Course-scoped: teachers manage assignments; everyone enrolled can read.
INSERT INTO authz_policies (resource, verb, course_role) VALUES
  ('assignment', 'create', 'teacher'),
  ('assignment', 'update', 'teacher'),
  ('assignment', 'update', 'assistant'),
  ('assignment', 'get',    'teacher'),
  ('assignment', 'get',    'assistant'),
  ('assignment', 'get',    'student'),
  ('assignment', 'get',    'observer'),
  ('assignment', 'delete', 'teacher');

--  Scope-based: viewers can read for oversight; admins can create / update /
--  delete for moderation or curriculum-wide assignments.
INSERT INTO authz_policies (resource, verb, scope_type, min_level) VALUES
  ('assignment', 'get',    'department', 'viewer'),
  ('assignment', 'get',    'university', 'viewer'),
  ('assignment', 'get',    'platform',   'admin'),
  ('assignment', 'create', 'department', 'admin'),
  ('assignment', 'create', 'college',    'admin'),
  ('assignment', 'create', 'university', 'admin'),
  ('assignment', 'create', 'platform',   'admin'),
  ('assignment', 'update', 'department', 'admin'),
  ('assignment', 'update', 'college',    'admin'),
  ('assignment', 'update', 'university', 'admin'),
  ('assignment', 'update', 'platform',   'admin'),
  ('assignment', 'delete', 'department', 'admin'),
  ('assignment', 'delete', 'college',    'admin'),
  ('assignment', 'delete', 'university', 'admin'),
  ('assignment', 'delete', 'platform',   'admin');

-- ── Exam ───────────────────────────────────────────────────────────────────────
--  Course-scoped: teachers create / update / delete; all enrolled can read.
INSERT INTO authz_policies (resource, verb, course_role) VALUES
  ('exam', 'create', 'teacher'),
  ('exam', 'get',    'teacher'),
  ('exam', 'get',    'assistant'),
  ('exam', 'get',    'student'),
  ('exam', 'get',    'observer'),
  ('exam', 'update', 'teacher'),
  ('exam', 'delete', 'teacher');

--  Scope-based: viewers can read; admins can create (used for question-bank
--  access scoped by course_code, not offering).
INSERT INTO authz_policies (resource, verb, scope_type, min_level) VALUES
  ('exam', 'get',    'department', 'viewer'),
  ('exam', 'get',    'university', 'viewer'),
  ('exam', 'get',    'platform',   'admin'),
  ('exam', 'create', 'department', 'admin'),
  ('exam', 'create', 'university', 'admin'),
  ('exam', 'create', 'platform',   'admin');

-- ── Grade ──────────────────────────────────────────────────────────────────────
--  Purely course-scoped.  Teachers record / update; all enrolled can read.
INSERT INTO authz_policies (resource, verb, course_role) VALUES
  ('grade', 'create', 'teacher'),
  ('grade', 'get',    'teacher'),
  ('grade', 'get',    'assistant'),
  ('grade', 'get',    'student'),
  ('grade', 'get',    'observer'),
  ('grade', 'update', 'teacher');

-- ── Project ────────────────────────────────────────────────────────────────────
--  Course-scoped: teachers manage; assistants can update; all enrolled read.
INSERT INTO authz_policies (resource, verb, course_role) VALUES
  ('project', 'create', 'teacher'),
  ('project', 'update', 'teacher'),
  ('project', 'update', 'assistant'),
  ('project', 'delete', 'teacher'),
  ('project', 'get',    'teacher'),
  ('project', 'get',    'assistant'),
  ('project', 'get',    'student'),
  ('project', 'get',    'observer');

--  Scope-based: viewers can read for oversight.
INSERT INTO authz_policies (resource, verb, scope_type, min_level) VALUES
  ('project', 'get',    'department', 'viewer'),
  ('project', 'get',    'university', 'viewer'),
  ('project', 'get',    'platform',   'admin');

-- ── Attendance ─────────────────────────────────────────────────────────────────
--  Course-scoped: teaching staff record and view attendance.
INSERT INTO authz_policies (resource, verb, course_role) VALUES
  ('attendance', 'update', 'teacher'),
  ('attendance', 'update', 'assistant'),
  ('attendance', 'get',    'teacher'),
  ('attendance', 'get',    'assistant');

--  Scope-based: admins can view attendance reports for oversight; they never
--  mutate attendance directly (that is a teaching task).
INSERT INTO authz_policies (resource, verb, scope_type, min_level) VALUES
  ('attendance', 'get',    'department', 'viewer'),
  ('attendance', 'get',    'university', 'viewer'),
  ('attendance', 'get',    'platform',   'admin');


-- ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
--  7. CURRICULUM, SETTINGS & BILLING
-- ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

-- ── Curriculum ─────────────────────────────────────────────────────────────────
--  Curriculum editing is admin-only, scoped program → platform.
INSERT INTO authz_policies (resource, verb, scope_type, min_level) VALUES
  ('curriculum', 'update', 'program',    'admin'),
  ('curriculum', 'update', 'department', 'admin'),
  ('curriculum', 'update', 'college',    'admin'),
  ('curriculum', 'update', 'university', 'admin'),
  ('curriculum', 'update', 'platform',   'admin');

-- ── Settings ───────────────────────────────────────────────────────────────────
INSERT INTO authz_policies (resource, verb, scope_type, min_level) VALUES
  ('settings', 'update', 'university', 'admin'),
  ('settings', 'update', 'platform',   'admin');

-- ── Subscription ───────────────────────────────────────────────────────────────
INSERT INTO authz_policies (resource, verb, scope_type, min_level) VALUES
  ('subscription', 'get',    'university', 'admin'),
  ('subscription', 'get',    'platform',   'admin'),
  ('subscription', 'update', 'platform',   'admin');


-- ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
--  8. POLICY MANAGEMENT  (super_admin only)
-- ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
--
--  Policy is the most sensitive resource.  Only super_admins can touch it,
--  and they can do so at every scope level so platform-level super_admins
--  can delegate university-level policy management to local super_admins.

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
