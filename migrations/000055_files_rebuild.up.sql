-- Files context rebuild: stored_files becomes inodes (content-addressed,
-- link-counted), user_files + folders unify into one files tree, and the
-- share/quota/media tables land. Design record: notes/files.md.
--
-- This migration never marks an inode 'gc': rows nothing references stay
-- 'live' with link_count 0, because a reference this backfill cannot see
-- (an external URL, a client bookmark) must err toward keeping bytes.
-- Physical objects keep their legacy keys (legacy_key); the re-key to
-- sha256/<hash> is a runtime job, not SQL.

-- ── Phase 1: merge duplicate content hashes ─────────────────────────────
-- The old upload path deduplicated by hash but check-then-act races could
-- slip duplicates through. The UNIQUE constraint below needs them merged:
-- the earliest row per hash wins, every referrer is repointed, losers die.
-- (Losers' physical objects become unreferenced; the re-key job reaps them.)

CREATE TEMP TABLE dup_inodes AS
SELECT id AS loser,
       first_value(id) OVER (PARTITION BY content_hash ORDER BY uploaded_at, id) AS canonical
FROM stored_files;
DELETE FROM dup_inodes WHERE loser = canonical;

UPDATE user_files t SET stored_file_id = d.canonical FROM dup_inodes d WHERE t.stored_file_id = d.loser;
UPDATE lesson_attachments t SET stored_file_id = d.canonical FROM dup_inodes d WHERE t.stored_file_id = d.loser;
UPDATE assignment_attachments t SET stored_file_id = d.canonical FROM dup_inodes d WHERE t.stored_file_id = d.loser;
UPDATE submission_files t SET stored_file_id = d.canonical FROM dup_inodes d WHERE t.stored_file_id = d.loser;
UPDATE post_attachments t SET stored_file_id = d.canonical FROM dup_inodes d WHERE t.stored_file_id = d.loser;
UPDATE activity_attachments t SET stored_file_id = d.canonical FROM dup_inodes d WHERE t.stored_file_id = d.loser;
UPDATE activities t SET cover_image_id = d.canonical FROM dup_inodes d WHERE t.cover_image_id = d.loser;
UPDATE project_attachments t SET stored_file_id = d.canonical FROM dup_inodes d WHERE t.stored_file_id = d.loser;
UPDATE project_submission_files t SET stored_file_id = d.canonical FROM dup_inodes d WHERE t.stored_file_id = d.loser;

DELETE FROM stored_files WHERE id IN (SELECT loser FROM dup_inodes);

-- ── Phase 2: stored_files → inodes ──────────────────────────────────────
-- Ownership leaves the inode (deduplicated bytes have no single owner;
-- ownership is a property of references). The storage key becomes a
-- nullable legacy pointer: rows claimed after this migration derive their
-- key from the hash and carry NULL here.

ALTER TABLE stored_files RENAME TO inodes;
ALTER TABLE inodes RENAME COLUMN uploaded_at TO created_at;
ALTER TABLE inodes RENAME COLUMN storage_key TO legacy_key;
ALTER TABLE inodes ALTER COLUMN legacy_key DROP NOT NULL;
ALTER TABLE inodes DROP COLUMN uploaded_by;

ALTER TABLE inodes
    ADD COLUMN link_count INT NOT NULL DEFAULT 0,
    ADD COLUMN state TEXT NOT NULL DEFAULT 'live',
    ADD CONSTRAINT inodes_link_count_nonneg CHECK (link_count >= 0),
    ADD CONSTRAINT inodes_size_nonneg CHECK (size_bytes >= 0),
    ADD CONSTRAINT inodes_state_valid CHECK (state IN ('live', 'gc'));

DROP INDEX IF EXISTS idx_stored_files_content_hash;
DROP INDEX IF EXISTS idx_stored_files_uploaded_by;
ALTER TABLE inodes ADD CONSTRAINT inodes_content_hash_key UNIQUE (content_hash);

-- The sweeper's candidate scan.
CREATE INDEX idx_inodes_gc ON inodes (id) WHERE state = 'gc';

-- ── Phase 3: user_files + folders → files ───────────────────────────────
-- One tree: a directory is a file with no inode (inode_id IS NULL). Duplicate
-- names are allowed by design — name is a label, id is the identity — so
-- the old unique name indexes are deliberately not recreated.
-- The parent_id self-FK is added after the copy: rows arrive in table
-- order, not tree order, and per-row FK checks would reject a child that
-- precedes its parent.

CREATE TABLE files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id UUID NOT NULL REFERENCES users(id),
    parent_id UUID,
    inode_id UUID REFERENCES inodes(id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    trashed BOOLEAN NOT NULL DEFAULT FALSE,
    explicitly_trashed BOOLEAN NOT NULL DEFAULT FALSE,
    trashed_at TIMESTAMPTZ,
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT files_name_length CHECK (length(name) BETWEEN 1 AND 255),
    CONSTRAINT files_trash_shape CHECK (explicitly_trashed = FALSE OR trashed = TRUE),
    CONSTRAINT files_trash_clock CHECK ((trashed_at IS NOT NULL) = trashed)
);

INSERT INTO files (id, owner_id, parent_id, inode_id, name, created_at, updated_at)
SELECT id, owner_id, parent_id, NULL, name, created_at, created_at
FROM folders;

INSERT INTO files (id, owner_id, parent_id, inode_id, name, created_at, updated_at)
SELECT id, owner_id, folder_id, stored_file_id, name, created_at, created_at
FROM user_files;

-- NO ACTION (not RESTRICT): the recursive delete removes a parent and its
-- children in one statement, and NO ACTION defers the check to statement
-- end where the whole subtree is already gone. The protection is the same —
-- a row can never be orphaned — without vetoing the one sanctioned way to
-- delete a subtree.
ALTER TABLE files
    ADD CONSTRAINT files_parent_fk FOREIGN KEY (parent_id) REFERENCES files(id);

DROP TABLE user_files;
DROP TABLE folders;

-- Listing (cursor order), tree walks, and unlink accounting.
CREATE INDEX idx_files_owner_parent_created ON files (owner_id, parent_id, created_at, id);
CREATE INDEX idx_files_parent ON files (parent_id);
CREATE INDEX idx_files_inode ON files (inode_id);

-- ── Phase 4: shares and quota ───────────────────────────────────────────

CREATE TABLE file_shares (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id UUID NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT file_shares_role_valid CHECK (role IN ('reader', 'writer')),
    CONSTRAINT file_shares_one_per_user UNIQUE (file_id, user_id)
);
CREATE INDEX idx_file_shares_user ON file_shares (user_id);

-- Maintained counter (never recomputed by scanning); seeded from the tree.
CREATE TABLE storage_usage (
    owner_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    bytes_used BIGINT NOT NULL DEFAULT 0,
    CONSTRAINT storage_usage_nonneg CHECK (bytes_used >= 0)
);
INSERT INTO storage_usage (owner_id, bytes_used)
SELECT f.owner_id, COALESCE(SUM(i.size_bytes), 0)
FROM files f
JOIN inodes i ON i.id = f.inode_id
GROUP BY f.owner_id;

-- ── Phase 5: link_count backfill ────────────────────────────────────────
-- Counts every reference the schema can see: the eight FK attachment
-- sites, activity covers, the files tree — plus UUIDs harvested from the
-- URL/JSONB columns that reference blobs without a foreign key (avatars,
-- logos, exam question images, application documents, settings). Those
-- string references are exactly why unreferenced rows stay 'live': the
-- harvest is best-effort protection, not proof of absence.

WITH refs AS (
    SELECT inode_id AS iid FROM files WHERE inode_id IS NOT NULL
    UNION ALL SELECT stored_file_id FROM lesson_attachments
    UNION ALL SELECT stored_file_id FROM assignment_attachments
    UNION ALL SELECT stored_file_id FROM submission_files
    UNION ALL SELECT stored_file_id FROM post_attachments
    UNION ALL SELECT stored_file_id FROM activity_attachments
    UNION ALL SELECT cover_image_id FROM activities WHERE cover_image_id IS NOT NULL
    UNION ALL SELECT stored_file_id FROM project_attachments
    UNION ALL SELECT stored_file_id FROM project_submission_files
    UNION ALL
    SELECT (m.token)::uuid
    FROM (
        SELECT avatar_url AS txt FROM users WHERE avatar_url IS NOT NULL
        UNION ALL SELECT logo_url FROM colleges WHERE logo_url IS NOT NULL
        UNION ALL SELECT logo_url FROM departments WHERE logo_url IS NOT NULL
        UNION ALL SELECT image_url FROM questions WHERE image_url IS NOT NULL
        UNION ALL SELECT documents::text FROM applications
        UNION ALL SELECT settings::text FROM settings
    ) sources,
    LATERAL (
        SELECT (regexp_matches(sources.txt,
            '[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}',
            'g'))[1] AS token
    ) m
    WHERE EXISTS (SELECT 1 FROM inodes i WHERE i.id = (m.token)::uuid)
)
UPDATE inodes i
SET link_count = c.n
FROM (SELECT iid, COUNT(*) AS n FROM refs GROUP BY iid) c
WHERE i.id = c.iid;

-- ── Phase 6: media sidecars (schema-complete, code comes later) ─────────

CREATE TABLE media (
    inode_id UUID PRIMARY KEY REFERENCES inodes(id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    state TEXT NOT NULL DEFAULT 'pending',
    duration_ms BIGINT,
    width INT,
    height INT,
    has_thumbnail BOOLEAN NOT NULL DEFAULT FALSE,
    has_storyboard BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT media_kind_valid CHECK (kind IN ('video', 'audio', 'image')),
    CONSTRAINT media_state_valid CHECK (state IN ('pending', 'processing', 'ready', 'failed'))
);

CREATE TABLE media_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    inode_id UUID NOT NULL REFERENCES inodes(id) ON DELETE CASCADE,
    state TEXT NOT NULL DEFAULT 'pending',
    attempts INT NOT NULL DEFAULT 0,
    last_error TEXT,
    heartbeat_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT media_jobs_state_valid CHECK (state IN ('pending', 'processing', 'ready', 'failed'))
);
CREATE INDEX idx_media_jobs_pending ON media_jobs (created_at) WHERE state = 'pending';

CREATE TABLE renditions (
    inode_id UUID NOT NULL REFERENCES inodes(id) ON DELETE CASCADE,
    quality TEXT NOT NULL,
    state TEXT NOT NULL DEFAULT 'pending',
    size_bytes BIGINT,
    PRIMARY KEY (inode_id, quality),
    CONSTRAINT renditions_quality_valid CHECK (quality IN ('1080p', '720p', '480p', '360p', '240p', 'audio')),
    CONSTRAINT renditions_state_valid CHECK (state IN ('pending', 'ready', 'failed'))
);
