// Package redis holds the Redis adapter for the identity domain: the
// refresh-token store of record.
package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/randdotdev/e-campus-server/internal/identity"
)

// tokenRepository stores refresh tokens in Redis (identity's TokenRepository
// port). Redis is the store of record here, not a cache: nothing repopulates
// it, and losing an entry ends that session. Keys expire with their token's
// TTL, so revoked and stale entries clean themselves up.
type tokenRepository struct {
	rdb *redis.Client
}

// NewTokenRepository wires the Redis-backed refresh-token store.
func NewTokenRepository(rdb *redis.Client) identity.TokenRepository {
	return &tokenRepository{rdb: rdb}
}

func tokenKey(hash string) string       { return fmt.Sprintf("token:%s", hash) }
func tokenUsedKey(hash string) string   { return fmt.Sprintf("token:used:%s", hash) }
func userTokensKey(id uuid.UUID) string { return fmt.Sprintf("user_tokens:%s", id) }
func familyKey(id uuid.UUID) string     { return fmt.Sprintf("family:%s", id) }

func (r *tokenRepository) CreateToken(ctx context.Context, token *identity.RefreshToken) error {
	token.ID = uuid.New()
	token.CreatedAt = time.Now()
	data, err := json.Marshal(token)
	if err != nil {
		return err
	}
	ttl := time.Until(token.ExpiresAt)
	if ttl <= 0 {
		return errors.New("token already expired")
	}
	pipe := r.rdb.Pipeline()
	pipe.Set(ctx, tokenKey(token.TokenHash), data, ttl)
	pipe.SAdd(ctx, userTokensKey(token.UserID), token.TokenHash)
	pipe.ExpireGT(ctx, userTokensKey(token.UserID), ttl)
	pipe.SAdd(ctx, familyKey(token.Family), token.TokenHash)
	pipe.ExpireGT(ctx, familyKey(token.Family), ttl)
	_, err = pipe.Exec(ctx)
	return err
}

func (r *tokenRepository) GetTokenByHash(ctx context.Context, hash string) (*identity.RefreshToken, error) {
	data, err := r.rdb.Get(ctx, tokenKey(hash)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, identity.ErrTokenNotFound
		}
		return nil, err
	}
	var token identity.RefreshToken
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, err
	}
	return &token, nil
}

// MarkTokenUsed flags the token as consumed via SetNX — an atomic
// compare-and-swap, so exactly one concurrent refresh wins and every other
// caller learns the token was already used (the reuse signal).
func (r *tokenRepository) MarkTokenUsed(ctx context.Context, hash string) (bool, error) {
	ttl := r.rdb.TTL(ctx, tokenKey(hash)).Val()
	if ttl <= 0 {
		return false, identity.ErrTokenNotFound
	}
	set, err := r.rdb.SetNX(ctx, tokenUsedKey(hash), "1", ttl).Result()
	if err != nil {
		return false, err
	}
	return !set, nil
}

func (r *tokenRepository) InvalidateFamily(ctx context.Context, family uuid.UUID) error {
	hashes, err := r.rdb.SMembers(ctx, familyKey(family)).Result()
	if err != nil {
		return err
	}
	if len(hashes) == 0 {
		return nil
	}
	userHashes := make(map[uuid.UUID][]string)
	for _, h := range hashes {
		token, err := r.GetTokenByHash(ctx, h)
		if err != nil {
			continue
		}
		userHashes[token.UserID] = append(userHashes[token.UserID], h)
	}
	pipe := r.rdb.Pipeline()
	for _, h := range hashes {
		pipe.Del(ctx, tokenKey(h))
		pipe.Del(ctx, tokenUsedKey(h))
	}
	for userID, hs := range userHashes {
		args := make([]any, len(hs))
		for i, h := range hs {
			args[i] = h
		}
		pipe.SRem(ctx, userTokensKey(userID), args...)
	}
	pipe.Del(ctx, familyKey(family))
	_, err = pipe.Exec(ctx)
	return err
}

func (r *tokenRepository) DeleteToken(ctx context.Context, hash string) error {
	token, err := r.GetTokenByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, identity.ErrTokenNotFound) {
			return nil
		}
		return err
	}
	pipe := r.rdb.Pipeline()
	pipe.Del(ctx, tokenKey(hash))
	pipe.Del(ctx, tokenUsedKey(hash))
	pipe.SRem(ctx, userTokensKey(token.UserID), hash)
	pipe.SRem(ctx, familyKey(token.Family), hash)
	_, err = pipe.Exec(ctx)
	return err
}

func (r *tokenRepository) DeleteUserTokens(ctx context.Context, userID uuid.UUID) error {
	hashes, err := r.rdb.SMembers(ctx, userTokensKey(userID)).Result()
	if err != nil {
		return err
	}
	if len(hashes) == 0 {
		return nil
	}
	pipe := r.rdb.Pipeline()
	for _, h := range hashes {
		pipe.Del(ctx, tokenKey(h))
		pipe.Del(ctx, tokenUsedKey(h))
	}
	pipe.Del(ctx, userTokensKey(userID))
	_, err = pipe.Exec(ctx)
	return err
}

func (r *tokenRepository) GetUserSessions(ctx context.Context, userID uuid.UUID) ([]identity.RefreshToken, error) {
	hashes, err := r.rdb.SMembers(ctx, userTokensKey(userID)).Result()
	if err != nil {
		return nil, err
	}
	var sessions []identity.RefreshToken
	for _, h := range hashes {
		token, err := r.GetTokenByHash(ctx, h)
		if err != nil {
			continue
		}
		if time.Now().Before(token.ExpiresAt) {
			sessions = append(sessions, *token)
		}
	}
	return sessions, nil
}
