DROP TABLE IF EXISTS user_files;
DROP TABLE IF EXISTS folders;
DROP TABLE IF EXISTS stored_files;

ALTER TABLE tier_limits
    DROP COLUMN IF EXISTS max_storage_bytes,
    DROP COLUMN IF EXISTS max_file_size_bytes;

ALTER TABLE subscription
    DROP COLUMN IF EXISTS max_storage_override,
    DROP COLUMN IF EXISTS max_file_size_override;

ALTER TABLE subscription_history
    DROP COLUMN IF EXISTS max_storage_override,
    DROP COLUMN IF EXISTS max_file_size_override;
