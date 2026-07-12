package postgres

import (
	"errors"

	"github.com/lib/pq"
)

// isUniqueViolation reports whether err is a PostgreSQL unique-constraint
// violation (code 23505). Callers translate it to the noun's already-exists
// sentinel.
func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505"
}
