package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ranjdotdev/e-campus-server/internal/config"
)

// MockUserStore implements UserStore interface for testing
type MockUserStore struct {
	CreateFunc      func(ctx context.Context, email, passwordHash, fullNameEN string, fullNameKU *string) (*UserData, error)
	GetByEmailFunc  func(ctx context.Context, email string) (*UserData, error)
	GetByIDFunc     func(ctx context.Context, id uuid.UUID) (*UserData, error)
	EmailExistsFunc func(ctx context.Context, email string) (bool, error)
	GetUserRoleFunc func(ctx context.Context, userID uuid.UUID) (*RoleData, error)
}

func (m *MockUserStore) Create(ctx context.Context, email, passwordHash, fullNameEN string, fullNameKU *string) (*UserData, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, email, passwordHash, fullNameEN, fullNameKU)
	}
	return &UserData{ID: uuid.New(), Email: email, FullNameEN: fullNameEN}, nil
}

func (m *MockUserStore) GetByEmail(ctx context.Context, email string) (*UserData, error) {
	if m.GetByEmailFunc != nil {
		return m.GetByEmailFunc(ctx, email)
	}
	return nil, ErrUserNotFound
}

func (m *MockUserStore) GetByID(ctx context.Context, id uuid.UUID) (*UserData, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, ErrUserNotFound
}

func (m *MockUserStore) EmailExists(ctx context.Context, email string) (bool, error) {
	if m.EmailExistsFunc != nil {
		return m.EmailExistsFunc(ctx, email)
	}
	return false, nil
}

func (m *MockUserStore) GetUserRole(ctx context.Context, userID uuid.UUID) (*RoleData, error) {
	if m.GetUserRoleFunc != nil {
		return m.GetUserRoleFunc(ctx, userID)
	}
	return nil, nil
}

// MockTokenRepository implements TokenRepository interface for testing
type MockTokenRepository struct {
	CreateTokenFunc      func(ctx context.Context, token *RefreshToken) error
	GetTokenByHashFunc   func(ctx context.Context, hash string) (*RefreshToken, error)
	MarkTokenUsedFunc    func(ctx context.Context, hash string) error
	InvalidateFamilyFunc func(ctx context.Context, family uuid.UUID) error
	DeleteTokenFunc      func(ctx context.Context, hash string) error
	DeleteUserTokensFunc func(ctx context.Context, userID uuid.UUID) error
	GetUserSessionsFunc  func(ctx context.Context, userID uuid.UUID) ([]RefreshToken, error)
}

func (m *MockTokenRepository) CreateToken(ctx context.Context, token *RefreshToken) error {
	if m.CreateTokenFunc != nil {
		return m.CreateTokenFunc(ctx, token)
	}
	token.ID = uuid.New()
	return nil
}

func (m *MockTokenRepository) GetTokenByHash(ctx context.Context, hash string) (*RefreshToken, error) {
	if m.GetTokenByHashFunc != nil {
		return m.GetTokenByHashFunc(ctx, hash)
	}
	return nil, ErrTokenNotFound
}

func (m *MockTokenRepository) MarkTokenUsed(ctx context.Context, hash string) error {
	if m.MarkTokenUsedFunc != nil {
		return m.MarkTokenUsedFunc(ctx, hash)
	}
	return nil
}

func (m *MockTokenRepository) InvalidateFamily(ctx context.Context, family uuid.UUID) error {
	if m.InvalidateFamilyFunc != nil {
		return m.InvalidateFamilyFunc(ctx, family)
	}
	return nil
}

func (m *MockTokenRepository) DeleteToken(ctx context.Context, hash string) error {
	if m.DeleteTokenFunc != nil {
		return m.DeleteTokenFunc(ctx, hash)
	}
	return nil
}

func (m *MockTokenRepository) DeleteUserTokens(ctx context.Context, userID uuid.UUID) error {
	if m.DeleteUserTokensFunc != nil {
		return m.DeleteUserTokensFunc(ctx, userID)
	}
	return nil
}

func (m *MockTokenRepository) GetUserSessions(ctx context.Context, userID uuid.UUID) ([]RefreshToken, error) {
	if m.GetUserSessionsFunc != nil {
		return m.GetUserSessionsFunc(ctx, userID)
	}
	return []RefreshToken{}, nil
}

func testJWTConfig() *config.JWTConfig {
	return &config.JWTConfig{
		Secret:     "test-secret-key-for-testing-purposes",
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 7 * 24 * time.Hour,
	}
}

func TestRegister_Success(t *testing.T) {
	userID := uuid.New()
	mockUsers := &MockUserStore{
		EmailExistsFunc: func(ctx context.Context, email string) (bool, error) {
			return false, nil
		},
		CreateFunc: func(ctx context.Context, email, passwordHash, fullNameEN string, fullNameKU *string) (*UserData, error) {
			return &UserData{
				ID:         userID,
				Email:      email,
				FullNameEN: fullNameEN,
				IsActive:   true,
			}, nil
		},
	}
	mockTokens := &MockTokenRepository{}

	svc := NewService(mockTokens, mockUsers, testJWTConfig())

	req := RegisterRequest{
		Email:      "test@example.com",
		Password:   "password123",
		FullNameEN: "Test User",
	}

	user, err := svc.Register(context.Background(), req)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if user.ID != userID {
		t.Errorf("user.ID = %v, want %v", user.ID, userID)
	}
	if user.Email != "test@example.com" {
		t.Errorf("user.Email = %v, want test@example.com", user.Email)
	}
}

func TestRegister_EmailExists(t *testing.T) {
	mockUsers := &MockUserStore{
		EmailExistsFunc: func(ctx context.Context, email string) (bool, error) {
			return true, nil
		},
	}
	mockTokens := &MockTokenRepository{}

	svc := NewService(mockTokens, mockUsers, testJWTConfig())

	req := RegisterRequest{
		Email:      "existing@example.com",
		Password:   "password123",
		FullNameEN: "Test User",
	}

	_, err := svc.Register(context.Background(), req)
	if !errors.Is(err, ErrEmailExists) {
		t.Errorf("Register() error = %v, want ErrEmailExists", err)
	}
}

func TestLogin_Success(t *testing.T) {
	userID := uuid.New()
	passwordHash, _ := HashPassword("password123")

	mockUsers := &MockUserStore{
		GetByEmailFunc: func(ctx context.Context, email string) (*UserData, error) {
			return &UserData{
				ID:           userID,
				Email:        email,
				PasswordHash: passwordHash,
				FullNameEN:   "Test User",
				IsActive:     true,
			}, nil
		},
		GetUserRoleFunc: func(ctx context.Context, userID uuid.UUID) (*RoleData, error) {
			return nil, nil
		},
	}
	mockTokens := &MockTokenRepository{
		CreateTokenFunc: func(ctx context.Context, token *RefreshToken) error {
			token.ID = uuid.New()
			return nil
		},
	}

	svc := NewService(mockTokens, mockUsers, testJWTConfig())

	req := LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}

	tokens, user, err := svc.Login(context.Background(), req, "Test Device", "127.0.0.1")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	if tokens.AccessToken == "" {
		t.Error("tokens.AccessToken is empty")
	}
	if tokens.RefreshToken == "" {
		t.Error("tokens.RefreshToken is empty")
	}
	if user.ID != userID {
		t.Errorf("user.ID = %v, want %v", user.ID, userID)
	}
}

func TestLogin_InvalidCredentials_UserNotFound(t *testing.T) {
	mockUsers := &MockUserStore{
		GetByEmailFunc: func(ctx context.Context, email string) (*UserData, error) {
			return nil, ErrUserNotFound
		},
	}
	mockTokens := &MockTokenRepository{}

	svc := NewService(mockTokens, mockUsers, testJWTConfig())

	req := LoginRequest{
		Email:    "nonexistent@example.com",
		Password: "password123",
	}

	_, _, err := svc.Login(context.Background(), req, "Test Device", "127.0.0.1")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("Login() error = %v, want ErrInvalidCredentials", err)
	}
}

func TestLogin_InvalidCredentials_WrongPassword(t *testing.T) {
	passwordHash, _ := HashPassword("correctpassword")

	mockUsers := &MockUserStore{
		GetByEmailFunc: func(ctx context.Context, email string) (*UserData, error) {
			return &UserData{
				ID:           uuid.New(),
				Email:        email,
				PasswordHash: passwordHash,
				IsActive:     true,
			}, nil
		},
	}
	mockTokens := &MockTokenRepository{}

	svc := NewService(mockTokens, mockUsers, testJWTConfig())

	req := LoginRequest{
		Email:    "test@example.com",
		Password: "wrongpassword",
	}

	_, _, err := svc.Login(context.Background(), req, "Test Device", "127.0.0.1")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("Login() error = %v, want ErrInvalidCredentials", err)
	}
}

func TestLogin_UserInactive(t *testing.T) {
	passwordHash, _ := HashPassword("password123")

	mockUsers := &MockUserStore{
		GetByEmailFunc: func(ctx context.Context, email string) (*UserData, error) {
			return &UserData{
				ID:           uuid.New(),
				Email:        email,
				PasswordHash: passwordHash,
				IsActive:     false,
			}, nil
		},
	}
	mockTokens := &MockTokenRepository{}

	svc := NewService(mockTokens, mockUsers, testJWTConfig())

	req := LoginRequest{
		Email:    "inactive@example.com",
		Password: "password123",
	}

	_, _, err := svc.Login(context.Background(), req, "Test Device", "127.0.0.1")
	if !errors.Is(err, ErrUserInactive) {
		t.Errorf("Login() error = %v, want ErrUserInactive", err)
	}
}

func TestRefresh_Success(t *testing.T) {
	userID := uuid.New()
	family := uuid.New()
	refreshToken := GenerateRefreshToken()
	tokenHash := HashToken(refreshToken)

	mockUsers := &MockUserStore{
		GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*UserData, error) {
			return &UserData{
				ID:         userID,
				Email:      "test@example.com",
				FullNameEN: "Test User",
				IsActive:   true,
			}, nil
		},
		GetUserRoleFunc: func(ctx context.Context, userID uuid.UUID) (*RoleData, error) {
			return nil, nil
		},
	}
	mockTokens := &MockTokenRepository{
		GetTokenByHashFunc: func(ctx context.Context, hash string) (*RefreshToken, error) {
			return &RefreshToken{
				ID:        uuid.New(),
				UserID:    userID,
				TokenHash: tokenHash,
				Family:    family,
				ExpiresAt: time.Now().Add(time.Hour),
				UsedAt:    nil,
			}, nil
		},
		MarkTokenUsedFunc: func(ctx context.Context, hash string) error {
			return nil
		},
		CreateTokenFunc: func(ctx context.Context, token *RefreshToken) error {
			token.ID = uuid.New()
			return nil
		},
	}

	svc := NewService(mockTokens, mockUsers, testJWTConfig())

	tokens, err := svc.Refresh(context.Background(), refreshToken, "Test Device", "127.0.0.1")
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}

	if tokens.AccessToken == "" {
		t.Error("tokens.AccessToken is empty")
	}
	if tokens.RefreshToken == "" {
		t.Error("tokens.RefreshToken is empty")
	}
}

func TestRefresh_TokenExpired(t *testing.T) {
	refreshToken := GenerateRefreshToken()

	mockUsers := &MockUserStore{}
	mockTokens := &MockTokenRepository{
		GetTokenByHashFunc: func(ctx context.Context, hash string) (*RefreshToken, error) {
			return &RefreshToken{
				ID:        uuid.New(),
				UserID:    uuid.New(),
				ExpiresAt: time.Now().Add(-time.Hour),
			}, nil
		},
	}

	svc := NewService(mockTokens, mockUsers, testJWTConfig())

	_, err := svc.Refresh(context.Background(), refreshToken, "Test Device", "127.0.0.1")
	if !errors.Is(err, ErrTokenExpired) {
		t.Errorf("Refresh() error = %v, want ErrTokenExpired", err)
	}
}

func TestRefresh_TokenReused(t *testing.T) {
	refreshToken := GenerateRefreshToken()
	usedAt := time.Now()

	mockUsers := &MockUserStore{}
	mockTokens := &MockTokenRepository{
		GetTokenByHashFunc: func(ctx context.Context, hash string) (*RefreshToken, error) {
			return &RefreshToken{
				ID:        uuid.New(),
				UserID:    uuid.New(),
				Family:    uuid.New(),
				ExpiresAt: time.Now().Add(time.Hour),
				UsedAt:    &usedAt,
			}, nil
		},
		InvalidateFamilyFunc: func(ctx context.Context, family uuid.UUID) error {
			return nil
		},
	}

	svc := NewService(mockTokens, mockUsers, testJWTConfig())

	_, err := svc.Refresh(context.Background(), refreshToken, "Test Device", "127.0.0.1")
	if !errors.Is(err, ErrTokenReused) {
		t.Errorf("Refresh() error = %v, want ErrTokenReused", err)
	}
}

func TestLogout_Success(t *testing.T) {
	deleteTokenCalled := false
	mockUsers := &MockUserStore{}
	mockTokens := &MockTokenRepository{
		DeleteTokenFunc: func(ctx context.Context, hash string) error {
			deleteTokenCalled = true
			return nil
		},
	}

	svc := NewService(mockTokens, mockUsers, testJWTConfig())

	err := svc.Logout(context.Background(), "some-refresh-token")
	if err != nil {
		t.Fatalf("Logout() error = %v", err)
	}

	if !deleteTokenCalled {
		t.Error("DeleteToken was not called")
	}
}

func TestLogoutAll_Success(t *testing.T) {
	userID := uuid.New()
	deleteUserTokensCalled := false

	mockUsers := &MockUserStore{}
	mockTokens := &MockTokenRepository{
		DeleteUserTokensFunc: func(ctx context.Context, id uuid.UUID) error {
			if id != userID {
				t.Errorf("DeleteUserTokens called with %v, want %v", id, userID)
			}
			deleteUserTokensCalled = true
			return nil
		},
	}

	svc := NewService(mockTokens, mockUsers, testJWTConfig())

	err := svc.LogoutAll(context.Background(), userID)
	if err != nil {
		t.Fatalf("LogoutAll() error = %v", err)
	}

	if !deleteUserTokensCalled {
		t.Error("DeleteUserTokens was not called")
	}
}

func TestValidateAccessToken_Success(t *testing.T) {
	userID := uuid.New()
	jwtCfg := testJWTConfig()
	expiresAt := time.Now().Add(15 * time.Minute)

	accessToken, err := GenerateAccessToken(userID, "test@example.com", nil, expiresAt, jwtCfg.Secret)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	mockUsers := &MockUserStore{}
	mockTokens := &MockTokenRepository{}

	svc := NewService(mockTokens, mockUsers, jwtCfg)

	claims, err := svc.ValidateAccessToken(accessToken)
	if err != nil {
		t.Fatalf("ValidateAccessToken() error = %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("claims.UserID = %v, want %v", claims.UserID, userID)
	}
	if claims.Email != "test@example.com" {
		t.Errorf("claims.Email = %v, want test@example.com", claims.Email)
	}
}

func TestValidateAccessToken_InvalidToken(t *testing.T) {
	mockUsers := &MockUserStore{}
	mockTokens := &MockTokenRepository{}

	svc := NewService(mockTokens, mockUsers, testJWTConfig())

	_, err := svc.ValidateAccessToken("invalid-token")
	if err == nil {
		t.Error("ValidateAccessToken() should return error for invalid token")
	}
}
