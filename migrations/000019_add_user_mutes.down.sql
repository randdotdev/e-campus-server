-- Revert posts scope_type
ALTER TABLE posts DROP CONSTRAINT posts_scope_check;
ALTER TABLE posts ADD CONSTRAINT posts_scope_check CHECK (
    (scope_type = 'university' AND scope_id IS NULL) OR
    (scope_type != 'university' AND scope_id IS NOT NULL)
);

ALTER TABLE posts DROP CONSTRAINT posts_scope_type_check;
ALTER TABLE posts ADD CONSTRAINT posts_scope_type_check
    CHECK (scope_type IN ('university', 'college', 'department', 'program'));

-- Drop user_mutes
DROP TABLE IF EXISTS user_mutes;
