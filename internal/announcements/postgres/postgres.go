// Package postgres holds the SQL adapters for the announcements domain. One
// file per noun; each adapter satisfies that noun's port(s) and translates
// driver errors into the domain's sentinels here, at the single persistence
// translation point.
package postgres

import (
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/randdotdev/e-campus-server/internal/announcements"
)

// isUniqueViolation reports whether err is a PostgreSQL unique-constraint
// violation (code 23505). Callers translate it to the noun's already-exists
// sentinel.
func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505"
}

// scanUpdatedVersion reads the version returned by an optimistic compare-and-
// swap update. No rows means the version guard did not match: the caller has
// already confirmed the row exists, so this is a lost race, ErrConflict.
func scanUpdatedVersion(row *sqlx.Row) (int64, error) {
	var version int64
	if err := row.Scan(&version); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, announcements.ErrConflict
		}
		return 0, err
	}
	return version, nil
}
