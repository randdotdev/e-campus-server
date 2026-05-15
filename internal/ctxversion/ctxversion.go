// Package ctxversion tracks per-user context version counters in Redis.
// It is a leaf package (no internal imports) so middleware can import it
// without creating import cycles.
package ctxversion

import (
	"context"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const defaultVersion int64 = 1

func key(userID uuid.UUID) string {
	return fmt.Sprintf("ctx:v:%s", userID)
}

// Bump increments the context version for a user after a role change.
func Bump(ctx context.Context, rdb *redis.Client, userID uuid.UUID) {
	_ = rdb.Incr(ctx, key(userID)).Err()
}

// Get returns the current context version for a user. Returns 1 if unset.
func Get(ctx context.Context, rdb *redis.Client, userID uuid.UUID) int64 {
	v, err := rdb.Get(ctx, key(userID)).Int64()
	if err != nil {
		return defaultVersion
	}
	return v
}

// Header formats a version integer for use in an HTTP header value.
func Header(v int64) string {
	return strconv.FormatInt(v, 10)
}
