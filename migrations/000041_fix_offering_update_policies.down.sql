DELETE FROM authz_policies
WHERE resource = 'offering' AND verb = 'update' AND course_role IN ('teacher', 'assistant');
