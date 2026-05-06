DELETE FROM authz_policies WHERE resource = 'attendance';
DELETE FROM authz_policies WHERE resource = 'policy';
DELETE FROM authz_policies WHERE resource = 'user' AND verb IN ('list', 'update', 'delete');
DELETE FROM authz_policies WHERE resource IN ('department', 'college', 'program', 'university', 'platform');
DELETE FROM authz_policies WHERE resource = 'enrollment' AND course_role IS NOT NULL;
DELETE FROM authz_policies WHERE resource = 'enrollment' AND scope_type IS NOT NULL;
DELETE FROM authz_policies WHERE resource = 'qa' AND course_role IS NOT NULL;
DELETE FROM authz_policies WHERE resource IN ('assignment', 'exam', 'grade', 'project', 'student');
DELETE FROM authz_policies WHERE resource = 'offering' AND verb IN ('get', 'update');
