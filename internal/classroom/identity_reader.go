package classroom

import (
	"context"

	"github.com/google/uuid"
)

// UserReader resolves a user's display name (identity context) — the seed
// for a team's default name.
type UserReader interface {
	UserName(ctx context.Context, userID uuid.UUID) (string, error)
}
