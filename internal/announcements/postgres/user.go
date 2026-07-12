package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/randdotdev/e-campus-server/internal/announcements"
)

// UserLookup is the SQL adapter resolving @mention usernames against the
// published users table.
type UserLookup struct {
	db *sqlx.DB
}

var _ announcements.UserLookup = (*UserLookup)(nil)

// NewUserLookup wires the mention-resolution adapter.
func NewUserLookup(db *sqlx.DB) *UserLookup {
	return &UserLookup{db: db}
}

func (r *UserLookup) GetUserIDsByUsernames(ctx context.Context, usernames []string) (map[string]uuid.UUID, error) {
	if len(usernames) == 0 {
		return map[string]uuid.UUID{}, nil
	}
	type row struct {
		ID       uuid.UUID `db:"id"`
		Username string    `db:"username"`
	}
	query, args, err := sqlx.In(`SELECT id, username FROM users WHERE username IN (?)`, usernames)
	if err != nil {
		return nil, err
	}
	query = r.db.Rebind(query)
	var rows []row
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, err
	}
	result := make(map[string]uuid.UUID, len(rows))
	for _, rw := range rows {
		result[rw.Username] = rw.ID
	}
	return result, nil
}
