-- Add missing course-role policies for offering/update.
-- All content mutations (sections, lessons, attachments, schedules) and
-- teacher management use authz.Check(ResourceOffering, ActionUpdate, offeringID).
-- The seed only had scope-based rows; teaching staff were never granted this verb.
INSERT INTO authz_policies (resource, verb, course_role) VALUES
  ('offering', 'update', 'teacher'),
  ('offering', 'update', 'assistant');
