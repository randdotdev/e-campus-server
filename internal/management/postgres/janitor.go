package postgres

import (
	"context"
	"log/slog"
	"time"

	"github.com/jmoiron/sqlx"
)

// Janitor permanently removes soft-deleted management rows once their
// recovery window has passed. Soft-deleted rows are invisible to every read
// but stay recoverable by hand until the purge; the janitor is what makes
// "deleted" eventually mean deleted.
type Janitor struct {
	db  *sqlx.DB
	log *slog.Logger
}

// NewJanitor wires the janitor.
func NewJanitor(db *sqlx.DB, log *slog.Logger) *Janitor {
	return &Janitor{db: db, log: log}
}

// Run purges once at boot and then daily, until the context ends. Rows are
// kept recoverable for 30 days after their soft delete.
func (j *Janitor) Run(ctx context.Context) {
	const retention = 30 * 24 * time.Hour
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for {
		purgeCtx, cancel := context.WithTimeout(ctx, time.Minute)
		if n, err := j.PurgeDeleted(purgeCtx, retention); err != nil {
			j.log.WarnContext(ctx, "management purge failed", "error", err)
		} else if n > 0 {
			j.log.InfoContext(ctx, "purged soft-deleted management rows", "rows", n)
		}
		cancel()

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// PurgeDeleted hard-deletes rows soft-deleted before the retention window.
// Children go first so foreign keys never block; enrollment rows attached to
// purged offerings are removed by the schema's ON DELETE CASCADE. It returns
// the number of rows purged and is safe to run concurrently — deletes are
// keyed on deleted_at, so two runs simply split the work.
func (j *Janitor) PurgeDeleted(ctx context.Context, retention time.Duration) (int64, error) {
	cutoff := time.Now().Add(-retention)
	var total int64
	for _, table := range []string{"course_offerings", "courses", "semesters"} {
		res, err := j.db.ExecContext(ctx,
			`DELETE FROM `+table+` WHERE deleted_at IS NOT NULL AND deleted_at < $1`, cutoff)
		if err != nil {
			return total, err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return total, err
		}
		total += n
	}
	return total, nil
}
