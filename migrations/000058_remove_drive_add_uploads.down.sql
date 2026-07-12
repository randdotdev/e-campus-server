ALTER INDEX inodes_pkey RENAME TO stored_files_pkey;
ALTER INDEX inodes_legacy_key_key RENAME TO stored_files_storage_key_key;

CREATE TABLE storage_usage (
    owner_id   uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    bytes_used bigint NOT NULL DEFAULT 0,
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT storage_usage_nonneg CHECK (bytes_used >= 0)
);

CREATE TABLE files (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id            uuid NOT NULL REFERENCES users(id),
    parent_id           uuid REFERENCES files(id),
    inode_id            uuid REFERENCES inodes(id) ON DELETE RESTRICT,
    name                text NOT NULL,
    trashed             boolean NOT NULL DEFAULT false,
    explicitly_trashed  boolean NOT NULL DEFAULT false,
    trashed_at          timestamptz,
    version             integer NOT NULL DEFAULT 1,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT files_name_length CHECK (length(name) >= 1 AND length(name) <= 255),
    CONSTRAINT files_trash_clock CHECK ((trashed_at IS NOT NULL) = trashed),
    CONSTRAINT files_trash_shape CHECK (explicitly_trashed = false OR trashed = true)
);

CREATE INDEX idx_files_owner_parent ON files (owner_id, parent_id) WHERE NOT trashed;
CREATE INDEX idx_files_inode ON files (inode_id);

CREATE TABLE file_shares (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id    uuid NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role       text NOT NULL,
    created_by uuid NOT NULL REFERENCES users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT file_shares_role_valid CHECK (role IN ('reader', 'writer')),
    CONSTRAINT file_shares_unique UNIQUE (file_id, user_id)
);

DROP TABLE uploads;
