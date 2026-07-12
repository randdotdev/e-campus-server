DROP INDEX IF EXISTS idx_authz_policies_permission;
ALTER TABLE authz_policies DROP CONSTRAINT IF EXISTS authz_policies_type_shape_check;
ALTER TABLE authz_policies DROP CONSTRAINT IF EXISTS authz_policies_type_check;
ALTER TABLE authz_policies DROP COLUMN IF EXISTS type;
ALTER TABLE authz_policies ADD CONSTRAINT authz_policies_resource_verb_scope_type_min_level_course_r_key
    UNIQUE (resource, verb, scope_type, min_level, course_role, domain);
