package authz

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

const cacheScanBatch = 500

type cache struct {
	rdb *redis.Client
}

func newCache(rdb *redis.Client) *cache {
	return &cache{rdb: rdb}
}

func (c *cache) get(ctx context.Context, key string, dest interface{}) bool {
	data, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(data), dest); err != nil {
		return false
	}
	return true
}

func (c *cache) set(ctx context.Context, key string, value interface{}, ttl time.Duration) {
	data, err := json.Marshal(value)
	if err != nil {
		return
	}
	_ = c.rdb.Set(ctx, key, data, ttl).Err()
}

func (c *cache) del(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return c.rdb.Unlink(ctx, keys...).Err()
}

func (c *cache) scanDel(ctx context.Context, pattern string) error {
	iter := c.rdb.Scan(ctx, 0, pattern, cacheScanBatch).Iterator()
	batch := make([]string, 0, cacheScanBatch)

	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		err := c.rdb.Unlink(ctx, batch...).Err()
		batch = batch[:0]
		return err
	}

	for iter.Next(ctx) {
		batch = append(batch, iter.Val())
		if len(batch) == cacheScanBatch {
			if err := flush(); err != nil {
				return err
			}
		}
	}
	if err := iter.Err(); err != nil {
		return err
	}
	return flush()
}
