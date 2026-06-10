-- Allow teaching staff to delete their own FAQs (knowledge-base entries
-- they authored). The service layer still enforces creator-only deletion
-- via CanDeleteQuestion; this just opens the authz gate for course role.
INSERT INTO authz_policies (resource, verb, course_role) VALUES
  ('qa', 'delete', 'teacher'),
  ('qa', 'delete', 'assistant');
