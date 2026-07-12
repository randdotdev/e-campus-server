DROP INDEX IF EXISTS idx_user_mutes_active_offering;
DROP INDEX IF EXISTS idx_user_mutes_active_university;

ALTER TABLE user_mutes DROP CONSTRAINT IF EXISTS user_mutes_scope_type_check;
ALTER TABLE user_mutes DROP CONSTRAINT IF EXISTS user_mutes_scope_check;

UPDATE user_mutes SET scope_type = 'course' WHERE scope_type = 'offering';

ALTER TABLE user_mutes
    ADD CONSTRAINT user_mutes_scope_type_check
    CHECK (scope_type IN ('course', 'university'));
ALTER TABLE user_mutes
    ADD CONSTRAINT user_mutes_scope_check CHECK (
        (scope_type = 'university' AND scope_id IS NULL) OR
        (scope_type = 'course' AND scope_id IS NOT NULL)
    );
