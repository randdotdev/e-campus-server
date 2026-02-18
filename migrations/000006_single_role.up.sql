-- Single role per user + add 'platform' scope type
ALTER TABLE roles
  DROP CONSTRAINT roles_user_id_permission_scope_type_scope_id_key,
  ADD CONSTRAINT roles_user_id_key UNIQUE (user_id),
  DROP CONSTRAINT roles_scope_type_check,
  ADD CONSTRAINT roles_scope_type_check CHECK (scope_type IN ('platform', 'university', 'college', 'department', 'program'));
