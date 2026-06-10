DELETE FROM authz_policies WHERE resource = 'offering' AND verb = 'get' AND course_role IN ('student', 'observer');
