CREATE TABLE authz_policies (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    resource    VARCHAR(50)  NOT NULL,
    verb        VARCHAR(20)  NOT NULL,
    scope_type  VARCHAR(20),
    min_level   VARCHAR(20),
    course_role VARCHAR(20),
    domain      VARCHAR(50),
    is_active   BOOLEAN      NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (resource, verb, scope_type, min_level, course_role, domain)
);

CREATE INDEX idx_authz_policies_lookup ON authz_policies (resource, verb) WHERE is_active = true;

-- ── Offering management ────────────────────────────────────────────────────
INSERT INTO authz_policies (resource, verb, scope_type, min_level) VALUES
  ('offering', 'create', 'department', 'admin'),
  ('offering', 'create', 'university', 'admin'),
  ('offering', 'create', 'platform',   'admin'),
  ('offering', 'update', 'department', 'admin'),
  ('offering', 'update', 'university', 'admin'),
  ('offering', 'update', 'platform',   'admin'),
  ('offering', 'delete', 'department', 'admin'),
  ('offering', 'delete', 'university', 'admin'),
  ('offering', 'delete', 'platform',   'admin');

-- ── Course (offering-scoped) ───────────────────────────────────────────────
INSERT INTO authz_policies (resource, verb, course_role) VALUES
  ('course', 'get',    'teacher'),
  ('course', 'get',    'assistant'),
  ('course', 'get',    'student'),
  ('course', 'get',    'observer'),
  ('course', 'update', 'teacher'),
  ('course', 'update', 'assistant');

INSERT INTO authz_policies (resource, verb, scope_type, min_level) VALUES
  ('course', 'get',    'program',    'viewer'),
  ('course', 'get',    'department', 'viewer'),
  ('course', 'get',    'university', 'viewer'),
  ('course', 'get',    'platform',   'admin'),
  ('course', 'create', 'department', 'admin'),
  ('course', 'create', 'university', 'admin'),
  ('course', 'create', 'platform',   'admin'),
  ('course', 'delete', 'department', 'admin'),
  ('course', 'delete', 'university', 'admin'),
  ('course', 'delete', 'platform',   'admin');

-- ── Academic calendar ──────────────────────────────────────────────────────
INSERT INTO authz_policies (resource, verb, scope_type, min_level) VALUES
  ('academic_year', 'get',    'university', 'viewer'),
  ('academic_year', 'get',    'platform',   'admin'),
  ('academic_year', 'list',   'university', 'viewer'),
  ('academic_year', 'list',   'platform',   'admin'),
  ('academic_year', 'create', 'university', 'admin'),
  ('academic_year', 'create', 'platform',   'admin'),
  ('academic_year', 'update', 'university', 'admin'),
  ('academic_year', 'update', 'platform',   'admin'),
  ('academic_year', 'delete', 'university', 'admin'),
  ('academic_year', 'delete', 'platform',   'admin'),
  ('semester',      'get',    'university', 'viewer'),
  ('semester',      'get',    'platform',   'admin'),
  ('semester',      'list',   'university', 'viewer'),
  ('semester',      'list',   'platform',   'admin'),
  ('semester',      'create', 'university', 'admin'),
  ('semester',      'create', 'platform',   'admin'),
  ('semester',      'update', 'university', 'admin'),
  ('semester',      'update', 'platform',   'admin'),
  ('semester',      'delete', 'university', 'admin'),
  ('semester',      'delete', 'platform',   'admin');

-- ── Students ───────────────────────────────────────────────────────────────
INSERT INTO authz_policies (resource, verb, scope_type, min_level) VALUES
  ('student', 'get',    'department', 'viewer'),
  ('student', 'get',    'university', 'viewer'),
  ('student', 'get',    'platform',   'admin'),
  ('student', 'list',   'department', 'viewer'),
  ('student', 'list',   'university', 'viewer'),
  ('student', 'list',   'platform',   'admin'),
  ('student', 'create', 'department', 'admin'),
  ('student', 'create', 'university', 'admin'),
  ('student', 'create', 'platform',   'admin'),
  ('student', 'update', 'department', 'admin'),
  ('student', 'update', 'university', 'admin'),
  ('student', 'update', 'platform',   'admin');

-- ── Enrollment ─────────────────────────────────────────────────────────────
INSERT INTO authz_policies (resource, verb, scope_type, min_level) VALUES
  ('enrollment', 'create', 'department', 'admin'),
  ('enrollment', 'create', 'university', 'admin'),
  ('enrollment', 'create', 'platform',   'admin'),
  ('enrollment', 'delete', 'department', 'admin'),
  ('enrollment', 'delete', 'university', 'admin'),
  ('enrollment', 'delete', 'platform',   'admin');

-- ── Exams ──────────────────────────────────────────────────────────────────
INSERT INTO authz_policies (resource, verb, course_role) VALUES
  ('exam', 'create', 'teacher'),   ('exam', 'create', 'assistant'),
  ('exam', 'update', 'teacher'),   ('exam', 'update', 'assistant'),
  ('exam', 'delete', 'teacher'),
  ('exam', 'get',    'teacher'),   ('exam', 'get',    'assistant'),
  ('exam', 'get',    'student');

-- ── Assignments ────────────────────────────────────────────────────────────
INSERT INTO authz_policies (resource, verb, course_role) VALUES
  ('assignment', 'create', 'teacher'),   ('assignment', 'create', 'assistant'),
  ('assignment', 'update', 'teacher'),   ('assignment', 'update', 'assistant'),
  ('assignment', 'delete', 'teacher'),
  ('assignment', 'get',    'teacher'),   ('assignment', 'get',    'assistant'),
  ('assignment', 'get',    'student'),
  ('assignment', 'submit', 'student');

-- ── Grades ─────────────────────────────────────────────────────────────────
INSERT INTO authz_policies (resource, verb, course_role) VALUES
  ('grade', 'create', 'teacher'),   ('grade', 'create', 'assistant'),
  ('grade', 'update', 'teacher'),   ('grade', 'update', 'assistant'),
  ('grade', 'get',    'teacher'),   ('grade', 'get',    'assistant'),
  ('grade', 'get',    'student');

-- ── Attendance ─────────────────────────────────────────────────────────────
INSERT INTO authz_policies (resource, verb, course_role) VALUES
  ('attendance', 'create', 'teacher'),   ('attendance', 'create', 'assistant'),
  ('attendance', 'update', 'teacher'),
  ('attendance', 'get',    'teacher'),   ('attendance', 'get',    'assistant'),
  ('attendance', 'get',    'student');

-- ── Users ──────────────────────────────────────────────────────────────────
INSERT INTO authz_policies (resource, verb, scope_type, min_level) VALUES
  ('user', 'list',   'university', 'admin'),
  ('user', 'list',   'platform',   'admin'),
  ('user', 'update', 'university', 'admin'),
  ('user', 'update', 'platform',   'admin'),
  ('user', 'delete', 'platform',   'admin');