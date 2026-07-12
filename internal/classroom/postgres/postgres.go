// Package postgres holds the SQL adapters for the classroom domain. One
// file per noun; each adapter satisfies that noun's port(s) and translates
// driver errors into the domain's sentinels here, at the single persistence
// translation point. readers.go additionally satisfies the cross-context
// reader ports with the published-table lookups §19a sanctions.
package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/randdotdev/e-campus-server/internal/classroom"
)

func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505"
}

func isForeignKeyViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23503"
}

// scanVersion reads the version returned by an optimistic compare-and-swap
// update. No rows means the version guard did not match: the caller already
// confirmed the row exists, so this is a lost race, ErrConflict.
func scanVersion(row *sqlx.Row) (int64, error) {
	var version int64
	if err := row.Scan(&version); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, classroom.ErrConflict
		}
		return 0, err
	}
	return version, nil
}

// inTx runs fn inside one transaction, translating panics into rollbacks.
func inTx(ctx context.Context, db *sqlx.DB, fn func(tx *sqlx.Tx) error) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}
