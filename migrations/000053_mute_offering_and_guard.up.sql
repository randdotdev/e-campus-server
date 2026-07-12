-- Communication mute revision: rename the offering-scope value to its true
-- noun, and add the write guard that closes the duplicate-mute race.

-- 1. Vocabulary: scope_id points at a course offering, so the scope is
--    'offering', not 'course'. Both CHECKs reference the old value, so drop
--    them, repoint existing rows, then re-add with 'offering'.
ALTER TABLE user_mutes DROP CONSTRAINT IF EXISTS user_mutes_scope_type_check;
ALTER TABLE user_mutes DROP CONSTRAINT IF EXISTS user_mutes_scope_check;

UPDATE user_mutes SET scope_type = 'offering' WHERE scope_type = 'course';

ALTER TABLE user_mutes
    ADD CONSTRAINT user_mutes_scope_type_check
    CHECK (scope_type IN ('offering', 'university'));
ALTER TABLE user_mutes
    ADD CONSTRAINT user_mutes_scope_check CHECK (
        (scope_type = 'university' AND scope_id IS NULL) OR
        (scope_type = 'offering' AND scope_id IS NOT NULL)
    );

-- 2. Concurrency (Shape 3): one open mute per user per scope. The service used
--    to read-then-insert (check-then-act), which two concurrent mutes could
--    both pass. These partial unique indexes make a second open mute impossible;
--    the adapter translates the violation to ErrAlreadyMuted. Split by scope
--    because scope_id is NULL for university-wide mutes (NULLs are distinct).
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_mutes_active_offering
    ON user_mutes(user_id, scope_id)
    WHERE unmuted_at IS NULL AND scope_type = 'offering';
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_mutes_active_university
    ON user_mutes(user_id)
    WHERE unmuted_at IS NULL AND scope_type = 'university';
