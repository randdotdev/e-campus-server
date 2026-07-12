-- The drive (personal file tree + shares + per-user quota) is removed as a
-- subsystem; attachments no longer route through it. Its replacement on the
-- attach path is the upload receipt: one counted reference proving who
-- brought the bytes, expired by the janitor if never attached.

CREATE TABLE uploads (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    inode_id    uuid NOT NULL REFERENCES inodes(id) ON DELETE RESTRICT,
    uploader_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        text NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uploads_name_length CHECK (length(name) >= 1 AND length(name) <= 255)
);

CREATE INDEX idx_uploads_uploader ON uploads (uploader_id);
CREATE INDEX idx_uploads_expiry ON uploads (created_at);

DROP TABLE file_shares;
DROP TABLE files;
DROP TABLE storage_usage;

-- Finish the 000055 rename: the table is inodes; its constraints should say so.
ALTER INDEX stored_files_pkey RENAME TO inodes_pkey;
ALTER INDEX stored_files_storage_key_key RENAME TO inodes_legacy_key_key;
