// Package postgres holds the SQL adapters for the management domain. One file
// per noun; each adapter satisfies that noun's port(s) and translates driver
// errors into the domain's sentinels here, at the single persistence
// translation point.
package postgres

import (
	"errors"

	"github.com/lib/pq"
)

// uniqueViolation is the PostgreSQL error code for a violated unique
// constraint — the signal that a constraint-guarded insert lost a race or
// duplicated an existing row.
const uniqueViolation = "23505"

// isUniqueViolation reports whether err is a PostgreSQL unique-constraint
// violation. Callers translate it to the noun's already-exists sentinel.
func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == uniqueViolation
}

// foreignKeyViolation is the PostgreSQL error code for a broken reference —
// the signal that a referenced row disappeared between a pre-check and the
// write.
const foreignKeyViolation = "23503"

// isForeignKeyViolation reports whether err is a PostgreSQL foreign-key
// violation. Callers translate it to the referenced noun's not-found
// sentinel.
func isForeignKeyViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == foreignKeyViolation
}
