package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/randdotdev/e-campus-server/internal/files"
)

type InodeRepository struct {
	db *sqlx.DB
}

var _ files.InodeRepository = (*InodeRepository)(nil)

func NewInodeRepository(db *sqlx.DB) *InodeRepository {
	return &InodeRepository{db: db}
}

func (r *InodeRepository) Get(ctx context.Context, id uuid.UUID) (*files.Inode, error) {
	var inode files.Inode
	err := r.db.GetContext(ctx, &inode, `SELECT * FROM inodes WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, files.ErrInodeNotFound
	}
	if err != nil {
		return nil, err
	}
	return &inode, nil
}

// Link is one atomic increment, guarded on the row still being live.
func (r *InodeRepository) Link(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE inodes SET link_count = link_count + 1 WHERE id = $1 AND state = 'live'`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		// Friendly message only; the guard above already decided.
		var exists bool
		if err := r.db.GetContext(ctx, &exists,
			`SELECT EXISTS (SELECT 1 FROM inodes WHERE id = $1)`, id); err != nil {
			return err
		}
		if exists {
			return files.ErrFileGone
		}
		return files.ErrInodeNotFound
	}
	return nil
}

// Unlink is one atomic decrement; zero marks the row for the sweeper in
// the same statement.
func (r *InodeRepository) Unlink(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE inodes
		SET link_count = link_count - 1,
		    state = CASE WHEN link_count - 1 = 0 THEN 'gc' ELSE state END
		WHERE id = $1 AND link_count >= 1`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return files.ErrInodeNotFound
	}
	return nil
}

func (r *InodeRepository) GCCandidates(ctx context.Context, limit int) ([]files.Inode, error) {
	var candidates []files.Inode
	err := r.db.SelectContext(ctx, &candidates,
		`SELECT * FROM inodes WHERE state = 'gc' LIMIT $1`, limit)
	return candidates, err
}

// Reclaim deletes one gc candidate iff still unreferenced, serialized per
// hash by the advisory lock. False means resurrected by a concurrent
// claim, or FK-vetoed and restored to live (the counter had drifted, and
// keeping bytes is the only safe answer).
func (r *InodeRepository) Reclaim(ctx context.Context, candidate files.Inode) (bool, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx,
		`SELECT pg_advisory_xact_lock(hashtext($1))`, candidate.ContentHash); err != nil {
		return false, err
	}

	res, err := tx.ExecContext(ctx,
		`DELETE FROM inodes WHERE id = $1 AND state = 'gc' AND link_count = 0`, candidate.ID)
	if isForeignKeyViolation(err) {
		// The FK backstop caught an uncounted reference; keep the row.
		_ = tx.Rollback()
		_, restoreErr := r.db.ExecContext(ctx,
			`UPDATE inodes SET state = 'live' WHERE id = $1`, candidate.ID)
		if restoreErr != nil {
			return false, fmt.Errorf("restore vetoed inode %s: %w", candidate.ID, restoreErr)
		}
		return false, fmt.Errorf("inode %s still referenced; restored to live", candidate.ID)
	}
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return n > 0, nil
}
