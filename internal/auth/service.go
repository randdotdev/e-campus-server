package auth

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/ranjdotdev/e-campus-server/internal/config"
)

type UserStore interface {
	Create(ctx context.Context, email, passwordHash, fullNameEN string, fullNameKU *string) (*UserData, error)
	GetByEmail(ctx context.Context, email string) (*UserData, error)
	GetByID(ctx context.Context, id uuid.UUID) (*UserData, error)
	EmailExists(ctx context.Context, email string) (bool, error)
	GetUserRole(ctx context.Context, userID uuid.UUID) (*RoleData, error)
}

type Service struct {
	tokens TokenRepository
	users  UserStore
	jwt    *config.JWTConfig
}

func NewService(tokens TokenRepository, users UserStore, jwt *config.JWTConfig) *Service {
	return &Service{tokens: tokens, users: users, jwt: jwt}
}

func (s *Service) Register(ctx context.Context, req RegisterRequest) (*UserData, error) {
	exists, err := s.users.EmailExists(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrEmailExists
	}

	hash, err := HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	return s.users.Create(ctx, req.Email, hash, req.FullNameEN, req.FullNameKU)
}

func (s *Service) Login(ctx context.Context, req LoginRequest, device, ip string) (*TokenPair, *UserData, error) {
	user, err := s.users.GetByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, nil, ErrInvalidCredentials
		}
		return nil, nil, err
	}

	if !user.IsActive {
		return nil, nil, ErrUserInactive
	}

	if !CheckPassword(req.Password, user.PasswordHash) {
		return nil, nil, ErrInvalidCredentials
	}

	tokens, err := s.createTokenPair(ctx, user, uuid.New(), device, ip)
	if err != nil {
		return nil, nil, err
	}

	return tokens, user, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken, device, ip string) (*TokenPair, error) {
	hash := HashToken(refreshToken)

	stored, err := s.tokens.GetTokenByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, ErrTokenNotFound) {
			return nil, ErrInvalidToken
		}
		return nil, err
	}

	if IsTokenExpired(stored.ExpiresAt) {
		return nil, ErrTokenExpired
	}

	if IsTokenUsed(stored.UsedAt) {
		_ = s.tokens.InvalidateFamily(ctx, stored.Family)
		return nil, ErrTokenReused
	}

	if err := s.tokens.MarkTokenUsed(ctx, hash); err != nil {
		return nil, err
	}

	user, err := s.users.GetByID(ctx, stored.UserID)
	if err != nil {
		return nil, err
	}

	return s.createTokenPair(ctx, user, stored.Family, device, ip)
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	hash := HashToken(refreshToken)
	return s.tokens.DeleteToken(ctx, hash)
}

func (s *Service) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	return s.tokens.DeleteUserTokens(ctx, userID)
}

func (s *Service) ValidateAccessToken(tokenString string) (*JWTClaims, error) {
	return ParseAccessToken(tokenString, s.jwt.Secret)
}

func (s *Service) createTokenPair(ctx context.Context, user *UserData, family uuid.UUID, device, ip string) (*TokenPair, error) {
	role, err := s.users.GetUserRole(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	expiresAt := time.Now().Add(s.jwt.AccessTTL)
	roleClaim := BuildRoleClaim(role)

	accessToken, err := GenerateAccessToken(user.ID, user.Email, roleClaim, expiresAt, s.jwt.Secret)
	if err != nil {
		return nil, err
	}

	refreshToken := GenerateRefreshToken()
	refreshHash := HashToken(refreshToken)

	dbToken := &RefreshToken{
		UserID:    user.ID,
		TokenHash: refreshHash,
		Family:    family,
		Device:    strPtr(device),
		IPAddress: strPtr(ip),
		ExpiresAt: time.Now().Add(s.jwt.RefreshTTL),
	}

	if err := s.tokens.CreateToken(ctx, dbToken); err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
	}, nil
}
