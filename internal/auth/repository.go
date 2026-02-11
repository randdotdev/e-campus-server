package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var (
	ErrTokenNotFound = errors.New("token not found")
)

type TokenRepository interface {
	CreateToken(ctx context.Context, token *RefreshToken) error
	GetTokenByHash(ctx context.Context, hash string) (*RefreshToken, error)
	MarkTokenUsed(ctx context.Context, id uuid.UUID) error
	InvalidateFamily(ctx context.Context, family uuid.UUID) error
	DeleteToken(ctx context.Context, hash string) error
	DeleteUserTokens(ctx context.Context, userID uuid.UUID) error
	GetUserSessions(ctx context.Context, userID uuid.UUID) ([]RefreshToken, error)
}

type tokenRepository struct {
	rdb *redis.Client
}

func NewTokenRepository(rdb *redis.Client) TokenRepository {
	return &tokenRepository{rdb: rdb}
}

func tokenKey(hash string) string {
	return fmt.Sprintf("token:%s", hash)
}

func userTokensKey(userID uuid.UUID) string {
	return fmt.Sprintf("user_tokens:%s", userID.String())
}

func familyKey(family uuid.UUID) string {
	return fmt.Sprintf("family:%s", family.String())
}

func (r *tokenRepository) CreateToken(ctx context.Context, token *RefreshToken) error {
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
	pipe.Expire(ctx, userTokensKey(token.UserID), ttl)
	pipe.SAdd(ctx, familyKey(token.Family), token.TokenHash)
	pipe.Expire(ctx, familyKey(token.Family), ttl)

	_, err = pipe.Exec(ctx)
	return err
}

func (r *tokenRepository) GetTokenByHash(ctx context.Context, hash string) (*RefreshToken, error) {
	data, err := r.rdb.Get(ctx, tokenKey(hash)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrTokenNotFound
		}
		return nil, err
	}

	var token RefreshToken
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, err
	}
	return &token, nil
}

func (r *tokenRepository) MarkTokenUsed(ctx context.Context, id uuid.UUID) error {
	// Scan for the token by ID - this is less efficient but maintains the interface
	// In practice, we have the hash when calling this, consider refactoring
	iter := r.rdb.Scan(ctx, 0, "token:*", 100).Iterator()
	for iter.Next(ctx) {
		data, err := r.rdb.Get(ctx, iter.Val()).Bytes()
		if err != nil {
			continue
		}

		var token RefreshToken
		if err := json.Unmarshal(data, &token); err != nil {
			continue
		}

		if token.ID == id {
			now := time.Now()
			token.UsedAt = &now
			newData, err := json.Marshal(token)
			if err != nil {
				return err
			}
			ttl := r.rdb.TTL(ctx, iter.Val()).Val()
			return r.rdb.Set(ctx, iter.Val(), newData, ttl).Err()
		}
	}
	return nil
}

func (r *tokenRepository) InvalidateFamily(ctx context.Context, family uuid.UUID) error {
	hashes, err := r.rdb.SMembers(ctx, familyKey(family)).Result()
	if err != nil {
		return err
	}

	if len(hashes) == 0 {
		return nil
	}

	pipe := r.rdb.Pipeline()
	for _, hash := range hashes {
		pipe.Del(ctx, tokenKey(hash))
	}
	pipe.Del(ctx, familyKey(family))
	_, err = pipe.Exec(ctx)
	return err
}

func (r *tokenRepository) DeleteToken(ctx context.Context, hash string) error {
	token, err := r.GetTokenByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, ErrTokenNotFound) {
			return nil
		}
		return err
	}

	pipe := r.rdb.Pipeline()
	pipe.Del(ctx, tokenKey(hash))
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
	for _, hash := range hashes {
		pipe.Del(ctx, tokenKey(hash))
	}
	pipe.Del(ctx, userTokensKey(userID))
	_, err = pipe.Exec(ctx)
	return err
}

func (r *tokenRepository) GetUserSessions(ctx context.Context, userID uuid.UUID) ([]RefreshToken, error) {
	hashes, err := r.rdb.SMembers(ctx, userTokensKey(userID)).Result()
	if err != nil {
		return nil, err
	}

	var sessions []RefreshToken
	for _, hash := range hashes {
		token, err := r.GetTokenByHash(ctx, hash)
		if err != nil {
			continue
		}
		if time.Now().Before(token.ExpiresAt) {
			sessions = append(sessions, *token)
		}
	}
	return sessions, nil
}
