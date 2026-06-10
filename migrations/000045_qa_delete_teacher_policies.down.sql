DELETE FROM authz_policies WHERE resource = 'qa' AND verb = 'delete' AND course_role IN ('teacher', 'assistant');
