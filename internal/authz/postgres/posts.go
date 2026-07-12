package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/authz"
)

// PostFacts satisfies authz.PostReader with one indexed read of the
// published posts table (§19a: author_id, scope_type, scope_id — read-only).
// A deleted post has no authority story: not found.
func (r *Readers) PostFacts(ctx context.Context, postID uuid.UUID) (authz.PostFacts, error) {
	var facts authz.PostFacts
	err := r.db.GetContext(ctx, &facts,
		`SELECT author_id, scope_type, scope_id FROM posts WHERE id = $1 AND deleted_at IS NULL`, postID)
	if errors.Is(err, sql.ErrNoRows) {
		return authz.PostFacts{}, authz.ErrTargetNotFound
	}
	if err != nil {
		return authz.PostFacts{}, err
	}
	return facts, nil
}

var _ authz.PostReader = (*Readers)(nil)
