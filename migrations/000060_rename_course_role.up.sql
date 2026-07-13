-- The posts scope value already migrated course→offering; this renames the
-- policy table's seat column to match. RENAME COLUMN rewrites the CHECK
-- constraints and the unique index expression with it.
ALTER TABLE authz_policies RENAME COLUMN course_role TO offering_role;
