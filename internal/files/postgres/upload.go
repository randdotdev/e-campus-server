package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/randdotdev/e-campus-server/internal/files"
)

// UploadRepository is the SQL adapter for upload receipts.
type UploadRepository struct {
	db *sqlx.DB
}

func NewUploadRepository(db *sqlx.DB) *UploadRepository {
	return &UploadRepository{db: db}
}

// CreateUpload claims the content in one transaction: inode upserted by
// hash (a dedup hit gains a link; a fresh hash starts at one) and the
// receipt inserted, serialized per hash by the advisory lock.
func (r *UploadRepository) CreateUpload(ctx context.Context, in files.UploadInput) (*files.Upload, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx,
		`SELECT pg_advisory_xact_lock(hashtext($1))`, in.ContentHash); err != nil {
		return nil, err
	}

	// On a dedup hit the existing row wins and gains a link; legacy_key
	// clears because the upload just wrote the content-addressed object.
	var inodeID uuid.UUID
	err = tx.GetContext(ctx, &inodeID, `
		INSERT INTO inodes (content_hash, size_bytes, mime_type, link_count, state)
		VALUES ($1, $2, $3, 1, 'live')
		ON CONFLICT (content_hash) DO UPDATE
		SET link_count = inodes.link_count + 1, state = 'live', legacy_key = NULL
		RETURNING id`,
		in.ContentHash, in.SizeBytes, in.MimeType)
	if err != nil {
		return nil, err
	}

	var up files.Upload
	err = tx.GetContext(ctx, &up, `
		INSERT INTO uploads (inode_id, uploader_id, name)
		VALUES ($1, $2, $3)
		RETURNING id, inode_id, uploader_id, name, created_at`,
		inodeID, in.UploaderID, in.Name)
	if err != nil {
		return nil, err
	}
	return &up, tx.Commit()
}

// GetContent returns the receipt with its inode's stored facts.
func (r *UploadRepository) GetContent(ctx context.Context, id uuid.UUID) (*files.Upload, *files.Inode, error) {
	var row struct {
		files.Upload
		Inode files.Inode `db:"inode"`
	}
	err := r.db.GetContext(ctx, &row, `
		SELECT u.id, u.inode_id, u.uploader_id, u.name, u.created_at,
		       i.id AS "inode.id", i.content_hash AS "inode.content_hash",
		       i.size_bytes AS "inode.size_bytes", i.mime_type AS "inode.mime_type",
		       i.link_count AS "inode.link_count", i.state AS "inode.state",
		       i.legacy_key AS "inode.legacy_key", i.created_at AS "inode.created_at"
		FROM uploads u
		JOIN inodes i ON i.id = u.inode_id
		WHERE u.id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, files.ErrUploadNotFound
	}
	if err != nil {
		return nil, nil, err
	}
	return &row.Upload, &row.Inode, nil
}

// ExpireBefore deletes receipts created before the cutoff, dropping each
// one's link in the same statement set — expired never-attached bytes
// become GC candidates.
func (r *UploadRepository) ExpireBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx, `
		WITH expired AS (
			DELETE FROM uploads WHERE created_at < $1
			RETURNING inode_id
		)
		UPDATE inodes i
		SET link_count = i.link_count - c.n,
		    state = CASE WHEN i.link_count - c.n = 0 THEN 'gc' ELSE i.state END
		FROM (SELECT inode_id, count(*) AS n FROM expired GROUP BY inode_id) c
		WHERE i.id = c.inode_id`, cutoff)
	if err != nil {
		return 0, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return n, tx.Commit()
}

var _ files.UploadRepository = (*UploadRepository)(nil)
