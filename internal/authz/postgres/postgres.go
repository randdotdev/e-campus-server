// Package postgres holds the PostgreSQL adapters for the authz context: the
// policy store (rows, seeding, reset) and the lineage/relation readers that
// resolve organisational ancestry and offering seats from published tables
// (§19a) owned by management and classroom.
package postgres

import (
	"errors"

	"github.com/lib/pq"
)

// isUniqueViolation reports whether err is a PostgreSQL unique-constraint
// violation (SQLSTATE 23505).
func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505"
}
