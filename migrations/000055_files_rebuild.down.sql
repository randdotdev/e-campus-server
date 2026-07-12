-- Reverses 000055. Shares, trash flags, versions, and media rows have no
-- pre-055 representation and are dropped; merged duplicate hashes are not
-- un-merged (the merge was semantically lossless).

DROP TABLE renditions;
DROP TABLE media_jobs;
DROP TABLE media;
DROP TABLE storage_usage;
DROP TABLE file_shares;

CREATE TABLE folders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    parent_id UUID REFERENCES folders(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_folders_owner_id ON folders(owner_id);
CREATE INDEX idx_folders_parent_id ON folders(parent_id);

CREATE TABLE user_files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    folder_id UUID REFERENCES folders(id) ON DELETE SET NULL,
    stored_file_id UUID NOT NULL REFERENCES inodes(id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_user_files_owner_id ON user_files(owner_id);
CREATE INDEX idx_user_files_folder_id ON user_files(folder_id);
CREATE INDEX idx_user_files_stored_file_id ON user_files(stored_file_id);
CREATE INDEX idx_user_files_owner_created ON user_files(owner_id, created_at DESC);

-- Folders first (parents before children within folders is not enforced by
-- FK during the copy because the constraint is created before the insert
-- here; disable and re-enable instead).
ALTER TABLE folders DROP CONSTRAINT folders_parent_id_fkey;
INSERT INTO folders (id, owner_id, parent_id, name, created_at)
SELECT id, owner_id, parent_id, name, created_at FROM files WHERE inode_id IS NULL;
ALTER TABLE folders ADD CONSTRAINT folders_parent_id_fkey
    FOREIGN KEY (parent_id) REFERENCES folders(id) ON DELETE CASCADE;

INSERT INTO user_files (id, owner_id, folder_id, stored_file_id, name, created_at)
SELECT id, owner_id, parent_id, inode_id, name, created_at FROM files WHERE inode_id IS NOT NULL;

DROP TABLE files;

DROP INDEX IF EXISTS idx_inodes_gc;
ALTER TABLE inodes DROP CONSTRAINT inodes_content_hash_key;
ALTER TABLE inodes DROP CONSTRAINT inodes_link_count_nonneg;
ALTER TABLE inodes DROP CONSTRAINT inodes_size_nonneg;
ALTER TABLE inodes DROP CONSTRAINT inodes_state_valid;
ALTER TABLE inodes DROP COLUMN link_count;
ALTER TABLE inodes DROP COLUMN state;
ALTER TABLE inodes ADD COLUMN uploaded_by UUID REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE inodes RENAME COLUMN legacy_key TO storage_key;
ALTER TABLE inodes RENAME COLUMN created_at TO uploaded_at;
ALTER TABLE inodes RENAME TO stored_files;

CREATE INDEX idx_stored_files_content_hash ON stored_files(content_hash);
CREATE INDEX idx_stored_files_uploaded_by ON stored_files(uploaded_by);
