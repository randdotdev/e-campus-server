-- authz: make the two permission shapes schema-authoritative (§16).
-- A permission row is exactly one type: 'staff' (min_level + scope_type,
-- optional domain) or 'offering' (course_role only — the seat column keeps its legacy name). The old table let the
-- shapes mix and let NULLs defeat the unique constraint.

-- 1. Remove rows that are malformed under the type model (never produced by
--    the seeds; defensive against hand edits).
DELETE FROM authz_policies
WHERE (course_role IS NULL AND (min_level IS NULL OR scope_type IS NULL))
   OR (course_role IS NOT NULL AND (min_level IS NOT NULL OR scope_type IS NOT NULL OR domain IS NOT NULL));

-- 2. Deduplicate: NULLs are distinct in the old UNIQUE constraint, so
--    re-applied policy migrations could have doubled rows. Keep the oldest.
DELETE FROM authz_policies a
USING authz_policies b
WHERE a.id > b.id
  AND a.resource = b.resource
  AND a.verb = b.verb
  AND COALESCE(a.scope_type, '')  = COALESCE(b.scope_type, '')
  AND COALESCE(a.min_level, '')   = COALESCE(b.min_level, '')
  AND COALESCE(a.course_role, '') = COALESCE(b.course_role, '')
  AND COALESCE(a.domain, '')      = COALESCE(b.domain, '');

-- 3. The type column, derived from the row's shape.
ALTER TABLE authz_policies ADD COLUMN type VARCHAR(10);
UPDATE authz_policies SET type = CASE WHEN course_role IS NOT NULL THEN 'offering' ELSE 'staff' END;
ALTER TABLE authz_policies ALTER COLUMN type SET NOT NULL;

-- 4. The shape rules, schema-authoritative.
ALTER TABLE authz_policies ADD CONSTRAINT authz_policies_type_check
    CHECK (type IN ('staff', 'offering'));
ALTER TABLE authz_policies ADD CONSTRAINT authz_policies_type_shape_check CHECK (
    (type = 'staff'  AND min_level IS NOT NULL AND scope_type IS NOT NULL AND course_role IS NULL)
 OR (type = 'offering' AND course_role IS NOT NULL AND min_level IS NULL AND scope_type IS NULL AND domain IS NULL)
);

-- 5. A unique index NULLs cannot defeat, replacing the old constraint
--    (dropped by whatever name Postgres generated for it).
DO $$
DECLARE name text;
BEGIN
    SELECT conname INTO name FROM pg_constraint
    WHERE conrelid = 'authz_policies'::regclass AND contype = 'u';
    IF name IS NOT NULL THEN
        EXECUTE format('ALTER TABLE authz_policies DROP CONSTRAINT %I', name);
    END IF;
END $$;
CREATE UNIQUE INDEX idx_authz_policies_permission ON authz_policies (
    resource, verb, type,
    COALESCE(scope_type, ''), COALESCE(min_level, ''),
    COALESCE(course_role, ''), COALESCE(domain, '')
);
