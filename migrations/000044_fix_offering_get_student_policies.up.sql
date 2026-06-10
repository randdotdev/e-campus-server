-- Add missing course-role policies for offering:get.
-- The seed had student and observer rows but they were never applied.
-- Without these, enrolled students and observers get 403 on GET /offerings/:id.
INSERT INTO authz_policies (resource, verb, course_role) VALUES
  ('offering', 'get', 'student'),
  ('offering', 'get', 'observer');
