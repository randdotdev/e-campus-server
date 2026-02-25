-- Stored files: actual S3 blob metadata (source of truth)
CREATE TABLE stored_files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    storage_key TEXT NOT NULL UNIQUE,
    content_hash TEXT NOT NULL,
    size_bytes BIGINT NOT NULL,
    mime_type TEXT NOT NULL,
    uploaded_by UUID REFERENCES users(id) ON DELETE SET NULL,
    uploaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_stored_files_content_hash ON stored_files(content_hash);
CREATE INDEX idx_stored_files_uploaded_by ON stored_files(uploaded_by);

-- Folders: user's personal organization
CREATE TABLE folders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    parent_id UUID REFERENCES folders(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_folders_owner_id ON folders(owner_id);
CREATE INDEX idx_folders_parent_id ON folders(parent_id);
CREATE UNIQUE INDEX idx_folders_unique_name ON folders(owner_id, parent_id, name)
    WHERE parent_id IS NOT NULL;
CREATE UNIQUE INDEX idx_folders_unique_root ON folders(owner_id, name)
    WHERE parent_id IS NULL;

-- User files: user's view of files in their storage
CREATE TABLE user_files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    folder_id UUID REFERENCES folders(id) ON DELETE SET NULL,
    stored_file_id UUID NOT NULL REFERENCES stored_files(id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_user_files_owner_id ON user_files(owner_id);
CREATE INDEX idx_user_files_folder_id ON user_files(folder_id);
CREATE INDEX idx_user_files_stored_file_id ON user_files(stored_file_id);
CREATE INDEX idx_user_files_owner_created ON user_files(owner_id, created_at DESC);

-- Storage limits in subscription
ALTER TABLE tier_limits
    ADD COLUMN max_storage_bytes BIGINT NOT NULL DEFAULT 5368709120,
    ADD COLUMN max_file_size_bytes BIGINT NOT NULL DEFAULT 104857600;

ALTER TABLE subscription
    ADD COLUMN max_storage_override BIGINT,
    ADD COLUMN max_file_size_override BIGINT;

ALTER TABLE subscription_history
    ADD COLUMN max_storage_override BIGINT,
    ADD COLUMN max_file_size_override BIGINT;

-- Update tier defaults (free: 5GB/100MB, basic: 50GB/200MB, premium: 500GB/1GB)
UPDATE tier_limits SET
    max_storage_bytes = 5368709120,
    max_file_size_bytes = 104857600
WHERE tier = 'free';

UPDATE tier_limits SET
    max_storage_bytes = 53687091200,
    max_file_size_bytes = 209715200
WHERE tier = 'basic';

UPDATE tier_limits SET
    max_storage_bytes = 536870912000,
    max_file_size_bytes = 1073741824
WHERE tier = 'premium';
