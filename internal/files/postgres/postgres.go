// Package postgres holds the SQL adapters for the files domain. One file
// per noun; each adapter satisfies that noun's port(s) and translates
// driver errors into the domain's sentinels here, at the single
// persistence translation point.
//
// The claim transaction and the sweeper's reclaim serialize per content
// hash with pg_advisory_xact_lock(hashtext(hash)) — the hash is locked as
// a concept because the row it names may not exist yet (first claim) or
// may just have died (reclaim racing a re-upload).
package postgres

import (
	"errors"

	"github.com/lib/pq"
)

// foreignKeyViolation is the PostgreSQL error code for a broken reference.
// In this context it is load-bearing twice: a claim naming a vanished
// parent, and — the belt-and-braces case — a reclaim trying to delete an
// inode some referrer table still points at despite a drifted counter.
const foreignKeyViolation = "23503"

// isForeignKeyViolation reports whether err is a PostgreSQL foreign-key
// violation.
func isForeignKeyViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == foreignKeyViolation
}
