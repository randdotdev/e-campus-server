// Package redis holds the caching decorator around the authz policy store.
// Policies are the one sanctioned cached keyspace (§21a): read on every
// request, changed only by rare admin writes. Deleting this decorator makes
// the system slower, never incorrect.
package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/randdotdev/e-campus-server/internal/authz"
)

const (
	policyTTL     = time.Hour
	keyPattern    = "authz:policy:*"
	scanBatchSize = 500
)

// PolicyCache decorates an authz.PolicyStore with cache-aside reads on
// PolicyFor and keyspace invalidation on every write. Cache failures are
// advisory: logged, then served from the source.
type PolicyCache struct {
	next authz.PolicyStore
	rdb  *redis.Client
	log  *slog.Logger
}

// NewPolicyCache wraps next with the Redis policy cache.
func NewPolicyCache(next authz.PolicyStore, rdb *redis.Client, log *slog.Logger) *PolicyCache {
	return &PolicyCache{next: next, rdb: rdb, log: log}
}

// PolicyFor is cache-aside: hit → return; miss → load from the store, set
// with TTL, return.
func (c *PolicyCache) PolicyFor(ctx context.Context, key authz.PolicyKey) (authz.Policy, error) {
	cacheKey := fmt.Sprintf("authz:policy:%s:%s", key.Resource, key.Action)

	if policy, ok := c.get(ctx, cacheKey); ok {
		return policy, nil
	}
	policy, err := c.next.PolicyFor(ctx, key)
	if err != nil {
		return authz.Policy{}, err
	}
	c.set(ctx, cacheKey, policy)
	return policy, nil
}

// ListPermissions is a pass-through: the admin list is not on the hot path.
func (c *PolicyCache) ListPermissions(ctx context.Context) ([]authz.Permission, error) {
	return c.next.ListPermissions(ctx)
}

// CreatePermission writes through and invalidates the pair's cache entry.
func (c *PolicyCache) CreatePermission(ctx context.Context, in authz.PermissionInput) (*authz.Permission, error) {
	created, err := c.next.CreatePermission(ctx, in)
	if err != nil {
		return nil, err
	}
	c.invalidate(ctx, fmt.Sprintf("authz:policy:%s:%s", in.Resource, in.Action))
	return created, nil
}

// DeactivatePermission writes through and clears the whole keyspace: the row
// id alone does not name its pair, and deactivation is rare admin-scale.
func (c *PolicyCache) DeactivatePermission(ctx context.Context, id uuid.UUID) error {
	if err := c.next.DeactivatePermission(ctx, id); err != nil {
		return err
	}
	c.invalidateAll(ctx)
	return nil
}

// Seed passes through and clears the keyspace so freshly seeded pairs are
// not shadowed by cached empty policies.
func (c *PolicyCache) Seed(ctx context.Context) error {
	if err := c.next.Seed(ctx); err != nil {
		return err
	}
	c.invalidateAll(ctx)
	return nil
}

// Reset passes through and clears the keyspace.
func (c *PolicyCache) Reset(ctx context.Context) error {
	if err := c.next.Reset(ctx); err != nil {
		return err
	}
	c.invalidateAll(ctx)
	return nil
}

func (c *PolicyCache) get(ctx context.Context, key string) (authz.Policy, bool) {
	data, err := c.rdb.Get(ctx, key).Bytes()
	if err != nil {
		return authz.Policy{}, false // miss or cache failure — same answer: go to source
	}
	var policy authz.Policy
	if err := json.Unmarshal(data, &policy); err != nil {
		return authz.Policy{}, false
	}
	return policy, true
}

func (c *PolicyCache) set(ctx context.Context, key string, policy authz.Policy) {
	data, err := json.Marshal(policy)
	if err != nil {
		return
	}
	if err := c.rdb.Set(ctx, key, data, policyTTL).Err(); err != nil {
		c.log.WarnContext(ctx, "authz: policy cache set failed", "key", key, "error", err)
	}
}

func (c *PolicyCache) invalidate(ctx context.Context, keys ...string) {
	if err := c.rdb.Unlink(ctx, keys...).Err(); err != nil {
		// TTL is the correctness backstop; a failed delete only extends staleness.
		c.log.WarnContext(ctx, "authz: policy cache invalidation failed", "keys", keys, "error", err)
	}
}

// invalidateAll pattern-scans the policy keyspace — reserved for admin-scale
// events (deactivate, seed, reset), never per-request paths.
func (c *PolicyCache) invalidateAll(ctx context.Context) {
	iter := c.rdb.Scan(ctx, 0, keyPattern, scanBatchSize).Iterator()
	batch := make([]string, 0, scanBatchSize)
	for iter.Next(ctx) {
		batch = append(batch, iter.Val())
		if len(batch) == scanBatchSize {
			c.invalidate(ctx, batch...)
			batch = batch[:0]
		}
	}
	if err := iter.Err(); err != nil {
		c.log.WarnContext(ctx, "authz: policy cache scan failed", "error", err)
	}
	if len(batch) > 0 {
		c.invalidate(ctx, batch...)
	}
}

var _ authz.PolicyStore = (*PolicyCache)(nil)
