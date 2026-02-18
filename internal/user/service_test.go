package user

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ranjdotdev/e-campus-server/internal/auth"
	"github.com/ranjdotdev/e-campus-server/internal/pagination"
)

// MockUserRepository implements UserRepository interface for testing
type MockUserRepository struct {
	GetUserFunc            func(ctx context.Context, id uuid.UUID) (*User, error)
	UpdateFunc             func(ctx context.Context, user *User) error
	UpdateEmailFunc        func(ctx context.Context, id uuid.UUID, email string) error
	EmailExistsFunc        func(ctx context.Context, email string) (bool, error)
	GetPasswordHashFunc    func(ctx context.Context, id uuid.UUID) (string, error)
	SetPasswordFunc        func(ctx context.Context, userID uuid.UUID, passwordHash string) error
	ListFunc               func(ctx context.Context, params pagination.PageParams, filters UserFilters) ([]User, bool, error)
	DeactivateFunc         func(ctx context.Context, id uuid.UUID) error
	GetRolesFunc           func(ctx context.Context, userID uuid.UUID) ([]Role, error)
	GetStaffProfileFunc    func(ctx context.Context, userID uuid.UUID) (*StaffProfile, error)
	CreateStaffProfileFunc func(ctx context.Context, profile *StaffProfile) error
	UpdateStaffProfileFunc func(ctx context.Context, profile *StaffProfile) error
	ScopeExistsFunc        func(ctx context.Context, scopeType string, scopeID uuid.UUID) (bool, error)
	CreateStaffUserTxFunc  func(ctx context.Context, user *User, profile *StaffProfile, role *Role) error
}

func (m *MockUserRepository) GetUser(ctx context.Context, id uuid.UUID) (*User, error) {
	if m.GetUserFunc != nil {
		return m.GetUserFunc(ctx, id)
	}
	return nil, ErrUserNotFound
}

func (m *MockUserRepository) Update(ctx context.Context, user *User) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, user)
	}
	return nil
}

func (m *MockUserRepository) UpdateEmail(ctx context.Context, id uuid.UUID, email string) error {
	if m.UpdateEmailFunc != nil {
		return m.UpdateEmailFunc(ctx, id, email)
	}
	return nil
}

func (m *MockUserRepository) EmailExists(ctx context.Context, email string) (bool, error) {
	if m.EmailExistsFunc != nil {
		return m.EmailExistsFunc(ctx, email)
	}
	return false, nil
}

func (m *MockUserRepository) GetPasswordHash(ctx context.Context, id uuid.UUID) (string, error) {
	if m.GetPasswordHashFunc != nil {
		return m.GetPasswordHashFunc(ctx, id)
	}
	return "", nil
}

func (m *MockUserRepository) SetPassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	if m.SetPasswordFunc != nil {
		return m.SetPasswordFunc(ctx, userID, passwordHash)
	}
	return nil
}

func (m *MockUserRepository) List(ctx context.Context, params pagination.PageParams, filters UserFilters) ([]User, bool, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, params, filters)
	}
	return []User{}, false, nil
}

func (m *MockUserRepository) Deactivate(ctx context.Context, id uuid.UUID) error {
	if m.DeactivateFunc != nil {
		return m.DeactivateFunc(ctx, id)
	}
	return nil
}

func (m *MockUserRepository) GetRoles(ctx context.Context, userID uuid.UUID) ([]Role, error) {
	if m.GetRolesFunc != nil {
		return m.GetRolesFunc(ctx, userID)
	}
	return []Role{}, nil
}

func (m *MockUserRepository) GetStaffProfile(ctx context.Context, userID uuid.UUID) (*StaffProfile, error) {
	if m.GetStaffProfileFunc != nil {
		return m.GetStaffProfileFunc(ctx, userID)
	}
	return nil, ErrStaffProfileNotFound
}

func (m *MockUserRepository) CreateStaffProfile(ctx context.Context, profile *StaffProfile) error {
	if m.CreateStaffProfileFunc != nil {
		return m.CreateStaffProfileFunc(ctx, profile)
	}
	profile.ID = uuid.New()
	return nil
}

func (m *MockUserRepository) UpdateStaffProfile(ctx context.Context, profile *StaffProfile) error {
	if m.UpdateStaffProfileFunc != nil {
		return m.UpdateStaffProfileFunc(ctx, profile)
	}
	return nil
}

func (m *MockUserRepository) ScopeExists(ctx context.Context, scopeType string, scopeID uuid.UUID) (bool, error) {
	if m.ScopeExistsFunc != nil {
		return m.ScopeExistsFunc(ctx, scopeType, scopeID)
	}
	return true, nil
}

func (m *MockUserRepository) CreateStaffUserTx(ctx context.Context, user *User, profile *StaffProfile, role *Role) error {
	if m.CreateStaffUserTxFunc != nil {
		return m.CreateStaffUserTxFunc(ctx, user, profile, role)
	}
	user.ID = uuid.New()
	profile.ID = uuid.New()
	if role != nil {
		role.ID = uuid.New()
	}
	return nil
}

// MockTokenRepository implements auth.TokenRepository interface for testing
type MockTokenRepository struct {
	GetUserSessionsFunc  func(ctx context.Context, userID uuid.UUID) ([]auth.RefreshToken, error)
	DeleteTokenFunc      func(ctx context.Context, hash string) error
	DeleteUserTokensFunc func(ctx context.Context, userID uuid.UUID) error
}

func (m *MockTokenRepository) CreateToken(ctx context.Context, token *auth.RefreshToken) error {
	return nil
}

func (m *MockTokenRepository) GetTokenByHash(ctx context.Context, hash string) (*auth.RefreshToken, error) {
	return nil, nil
}

func (m *MockTokenRepository) MarkTokenUsed(ctx context.Context, hash string) error {
	return nil
}

func (m *MockTokenRepository) InvalidateFamily(ctx context.Context, family uuid.UUID) error {
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

func (m *MockTokenRepository) GetUserSessions(ctx context.Context, userID uuid.UUID) ([]auth.RefreshToken, error) {
	if m.GetUserSessionsFunc != nil {
		return m.GetUserSessionsFunc(ctx, userID)
	}
	return []auth.RefreshToken{}, nil
}

func TestGetProfile_Success(t *testing.T) {
	userID := uuid.New()
	mockRepo := &MockUserRepository{
		GetUserFunc: func(ctx context.Context, id uuid.UUID) (*User, error) {
			return &User{
				ID:         userID,
				Email:      "test@example.com",
				FullNameEN: "Test User",
				IsActive:   true,
			}, nil
		},
	}
	mockTokens := &MockTokenRepository{}

	svc := NewService(mockRepo, mockTokens)

	user, err := svc.GetProfile(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetProfile() error = %v", err)
	}

	if user.ID != userID {
		t.Errorf("user.ID = %v, want %v", user.ID, userID)
	}
}

func TestGetProfile_NotFound(t *testing.T) {
	mockRepo := &MockUserRepository{
		GetUserFunc: func(ctx context.Context, id uuid.UUID) (*User, error) {
			return nil, ErrUserNotFound
		},
	}
	mockTokens := &MockTokenRepository{}

	svc := NewService(mockRepo, mockTokens)

	_, err := svc.GetProfile(context.Background(), uuid.New())
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("GetProfile() error = %v, want ErrUserNotFound", err)
	}
}

func TestUpdateProfile_Success(t *testing.T) {
	userID := uuid.New()
	newName := "Updated Name"

	mockRepo := &MockUserRepository{
		GetUserFunc: func(ctx context.Context, id uuid.UUID) (*User, error) {
			return &User{
				ID:         userID,
				Email:      "test@example.com",
				FullNameEN: "Original Name",
			}, nil
		},
		UpdateFunc: func(ctx context.Context, user *User) error {
			if user.FullNameEN != newName {
				t.Errorf("user.FullNameEN = %v, want %v", user.FullNameEN, newName)
			}
			return nil
		},
	}
	mockTokens := &MockTokenRepository{}

	svc := NewService(mockRepo, mockTokens)

	user, err := svc.UpdateProfile(context.Background(), userID, UpdateProfileRequest{
		FullNameEN: &newName,
	})
	if err != nil {
		t.Fatalf("UpdateProfile() error = %v", err)
	}

	if user.FullNameEN != newName {
		t.Errorf("user.FullNameEN = %v, want %v", user.FullNameEN, newName)
	}
}

func TestUpdateEmail_Success(t *testing.T) {
	userID := uuid.New()
	passwordHash, _ := auth.HashPassword("password123")

	mockRepo := &MockUserRepository{
		GetUserFunc: func(ctx context.Context, id uuid.UUID) (*User, error) {
			return &User{
				ID:    userID,
				Email: "old@example.com",
			}, nil
		},
		GetPasswordHashFunc: func(ctx context.Context, id uuid.UUID) (string, error) {
			return passwordHash, nil
		},
		EmailExistsFunc: func(ctx context.Context, email string) (bool, error) {
			return false, nil
		},
		UpdateEmailFunc: func(ctx context.Context, id uuid.UUID, email string) error {
			return nil
		},
	}
	mockTokens := &MockTokenRepository{}

	svc := NewService(mockRepo, mockTokens)

	err := svc.UpdateEmail(context.Background(), userID, UpdateEmailRequest{
		Email:    "new@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("UpdateEmail() error = %v", err)
	}
}

func TestUpdateEmail_SameEmail(t *testing.T) {
	userID := uuid.New()

	mockRepo := &MockUserRepository{
		GetUserFunc: func(ctx context.Context, id uuid.UUID) (*User, error) {
			return &User{
				ID:    userID,
				Email: "same@example.com",
			}, nil
		},
	}
	mockTokens := &MockTokenRepository{}

	svc := NewService(mockRepo, mockTokens)

	err := svc.UpdateEmail(context.Background(), userID, UpdateEmailRequest{
		Email:    "same@example.com",
		Password: "password123",
	})
	if !errors.Is(err, ErrSameEmail) {
		t.Errorf("UpdateEmail() error = %v, want ErrSameEmail", err)
	}
}

func TestUpdateEmail_InvalidPassword(t *testing.T) {
	userID := uuid.New()
	passwordHash, _ := auth.HashPassword("correctpassword")

	mockRepo := &MockUserRepository{
		GetUserFunc: func(ctx context.Context, id uuid.UUID) (*User, error) {
			return &User{
				ID:    userID,
				Email: "old@example.com",
			}, nil
		},
		GetPasswordHashFunc: func(ctx context.Context, id uuid.UUID) (string, error) {
			return passwordHash, nil
		},
	}
	mockTokens := &MockTokenRepository{}

	svc := NewService(mockRepo, mockTokens)

	err := svc.UpdateEmail(context.Background(), userID, UpdateEmailRequest{
		Email:    "new@example.com",
		Password: "wrongpassword",
	})
	if !errors.Is(err, ErrInvalidPassword) {
		t.Errorf("UpdateEmail() error = %v, want ErrInvalidPassword", err)
	}
}

func TestUpdateEmail_EmailExists(t *testing.T) {
	userID := uuid.New()
	passwordHash, _ := auth.HashPassword("password123")

	mockRepo := &MockUserRepository{
		GetUserFunc: func(ctx context.Context, id uuid.UUID) (*User, error) {
			return &User{
				ID:    userID,
				Email: "old@example.com",
			}, nil
		},
		GetPasswordHashFunc: func(ctx context.Context, id uuid.UUID) (string, error) {
			return passwordHash, nil
		},
		EmailExistsFunc: func(ctx context.Context, email string) (bool, error) {
			return true, nil
		},
	}
	mockTokens := &MockTokenRepository{}

	svc := NewService(mockRepo, mockTokens)

	err := svc.UpdateEmail(context.Background(), userID, UpdateEmailRequest{
		Email:    "existing@example.com",
		Password: "password123",
	})
	if !errors.Is(err, ErrEmailExists) {
		t.Errorf("UpdateEmail() error = %v, want ErrEmailExists", err)
	}
}

func TestGetSessions_Success(t *testing.T) {
	userID := uuid.New()
	sessionID := uuid.New()
	device := "Test Device"
	ip := "127.0.0.1"

	mockRepo := &MockUserRepository{}
	mockTokens := &MockTokenRepository{
		GetUserSessionsFunc: func(ctx context.Context, uid uuid.UUID) ([]auth.RefreshToken, error) {
			return []auth.RefreshToken{
				{
					ID:        sessionID,
					UserID:    userID,
					Device:    &device,
					IPAddress: &ip,
					CreatedAt: time.Now(),
					ExpiresAt: time.Now().Add(time.Hour),
				},
			}, nil
		},
	}

	svc := NewService(mockRepo, mockTokens)

	sessions, err := svc.GetSessions(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetSessions() error = %v", err)
	}

	if len(sessions) != 1 {
		t.Errorf("len(sessions) = %v, want 1", len(sessions))
	}
	if sessions[0].ID != sessionID {
		t.Errorf("sessions[0].ID = %v, want %v", sessions[0].ID, sessionID)
	}
}

func TestRevokeSession_Success(t *testing.T) {
	userID := uuid.New()
	sessionID := uuid.New()
	tokenHash := "test-hash"
	deleteTokenCalled := false

	mockRepo := &MockUserRepository{}
	mockTokens := &MockTokenRepository{
		GetUserSessionsFunc: func(ctx context.Context, uid uuid.UUID) ([]auth.RefreshToken, error) {
			return []auth.RefreshToken{
				{
					ID:        sessionID,
					UserID:    userID,
					TokenHash: tokenHash,
				},
			}, nil
		},
		DeleteTokenFunc: func(ctx context.Context, hash string) error {
			if hash != tokenHash {
				t.Errorf("DeleteToken called with %v, want %v", hash, tokenHash)
			}
			deleteTokenCalled = true
			return nil
		},
	}

	svc := NewService(mockRepo, mockTokens)

	err := svc.RevokeSession(context.Background(), userID, sessionID)
	if err != nil {
		t.Fatalf("RevokeSession() error = %v", err)
	}

	if !deleteTokenCalled {
		t.Error("DeleteToken was not called")
	}
}

func TestRevokeSession_NotFound(t *testing.T) {
	userID := uuid.New()

	mockRepo := &MockUserRepository{}
	mockTokens := &MockTokenRepository{
		GetUserSessionsFunc: func(ctx context.Context, uid uuid.UUID) ([]auth.RefreshToken, error) {
			return []auth.RefreshToken{}, nil
		},
	}

	svc := NewService(mockRepo, mockTokens)

	err := svc.RevokeSession(context.Background(), userID, uuid.New())
	if !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("RevokeSession() error = %v, want ErrSessionNotFound", err)
	}
}

func TestCreateStaffProfile_Success(t *testing.T) {
	userID := uuid.New()
	degree := "phd"

	mockRepo := &MockUserRepository{
		GetUserFunc: func(ctx context.Context, id uuid.UUID) (*User, error) {
			return &User{ID: userID}, nil
		},
		GetStaffProfileFunc: func(ctx context.Context, uid uuid.UUID) (*StaffProfile, error) {
			return nil, ErrStaffProfileNotFound
		},
		CreateStaffProfileFunc: func(ctx context.Context, profile *StaffProfile) error {
			profile.ID = uuid.New()
			return nil
		},
	}
	mockTokens := &MockTokenRepository{}

	svc := NewService(mockRepo, mockTokens)

	profile, err := svc.CreateStaffProfile(context.Background(), userID, UpdateStaffProfileRequest{
		HighestDegree: &degree,
	})
	if err != nil {
		t.Fatalf("CreateStaffProfile() error = %v", err)
	}

	if profile.HighestDegree == nil || *profile.HighestDegree != degree {
		t.Errorf("profile.HighestDegree = %v, want %v", profile.HighestDegree, degree)
	}
}

func TestCreateStaffProfile_AlreadyExists(t *testing.T) {
	userID := uuid.New()

	mockRepo := &MockUserRepository{
		GetUserFunc: func(ctx context.Context, id uuid.UUID) (*User, error) {
			return &User{ID: userID}, nil
		},
		GetStaffProfileFunc: func(ctx context.Context, uid uuid.UUID) (*StaffProfile, error) {
			return &StaffProfile{ID: uuid.New()}, nil
		},
	}
	mockTokens := &MockTokenRepository{}

	svc := NewService(mockRepo, mockTokens)

	_, err := svc.CreateStaffProfile(context.Background(), userID, UpdateStaffProfileRequest{})
	if !errors.Is(err, ErrStaffProfileExists) {
		t.Errorf("CreateStaffProfile() error = %v, want ErrStaffProfileExists", err)
	}
}

func TestChangePassword_Success(t *testing.T) {
	userID := uuid.New()
	currentHash, _ := auth.HashPassword("currentpassword")
	deleteUserTokensCalled := false

	mockRepo := &MockUserRepository{
		GetPasswordHashFunc: func(ctx context.Context, id uuid.UUID) (string, error) {
			return currentHash, nil
		},
		SetPasswordFunc: func(ctx context.Context, uid uuid.UUID, hash string) error {
			return nil
		},
	}
	mockTokens := &MockTokenRepository{
		DeleteUserTokensFunc: func(ctx context.Context, uid uuid.UUID) error {
			deleteUserTokensCalled = true
			return nil
		},
	}

	svc := NewService(mockRepo, mockTokens)

	err := svc.ChangePassword(context.Background(), userID, ChangePasswordRequest{
		CurrentPassword: "currentpassword",
		NewPassword:     "newpassword123",
	})
	if err != nil {
		t.Fatalf("ChangePassword() error = %v", err)
	}

	if !deleteUserTokensCalled {
		t.Error("DeleteUserTokens was not called")
	}
}

func TestChangePassword_InvalidPassword(t *testing.T) {
	userID := uuid.New()
	currentHash, _ := auth.HashPassword("correctpassword")

	mockRepo := &MockUserRepository{
		GetPasswordHashFunc: func(ctx context.Context, id uuid.UUID) (string, error) {
			return currentHash, nil
		},
	}
	mockTokens := &MockTokenRepository{}

	svc := NewService(mockRepo, mockTokens)

	err := svc.ChangePassword(context.Background(), userID, ChangePasswordRequest{
		CurrentPassword: "wrongpassword",
		NewPassword:     "newpassword123",
	})
	if !errors.Is(err, ErrInvalidPassword) {
		t.Errorf("ChangePassword() error = %v, want ErrInvalidPassword", err)
	}
}

func TestChangePassword_SamePassword(t *testing.T) {
	userID := uuid.New()
	currentHash, _ := auth.HashPassword("samepassword")

	mockRepo := &MockUserRepository{
		GetPasswordHashFunc: func(ctx context.Context, id uuid.UUID) (string, error) {
			return currentHash, nil
		},
	}
	mockTokens := &MockTokenRepository{}

	svc := NewService(mockRepo, mockTokens)

	err := svc.ChangePassword(context.Background(), userID, ChangePasswordRequest{
		CurrentPassword: "samepassword",
		NewPassword:     "samepassword",
	})
	if !errors.Is(err, ErrSamePassword) {
		t.Errorf("ChangePassword() error = %v, want ErrSamePassword", err)
	}
}

func TestCreateStaffUser_Success(t *testing.T) {
	adminID := uuid.New()
	actorRoles := []auth.RoleClaim{{Permission: "admin", ScopeType: "university"}}

	mockRepo := &MockUserRepository{
		EmailExistsFunc: func(ctx context.Context, email string) (bool, error) {
			return false, nil
		},
		CreateStaffUserTxFunc: func(ctx context.Context, user *User, profile *StaffProfile, role *Role) error {
			user.ID = uuid.New()
			profile.ID = uuid.New()
			return nil
		},
	}
	mockTokens := &MockTokenRepository{}

	svc := NewService(mockRepo, mockTokens)

	user, profile, _, err := svc.CreateStaffUser(context.Background(), adminID, actorRoles, CreateStaffUserRequest{
		Email:        "new@example.com",
		Password:     "password123",
		FullNameEN:   "New Staff",
		StaffProfile: UpdateStaffProfileRequest{},
	})
	if err != nil {
		t.Fatalf("CreateStaffUser() error = %v", err)
	}

	if user.Email != "new@example.com" {
		t.Errorf("user.Email = %v, want new@example.com", user.Email)
	}
	if profile == nil {
		t.Error("profile should not be nil")
	}
}

func TestCreateStaffUser_EmailExists(t *testing.T) {
	actorRoles := []auth.RoleClaim{{Permission: "admin", ScopeType: "university"}}
	mockRepo := &MockUserRepository{
		EmailExistsFunc: func(ctx context.Context, email string) (bool, error) {
			return true, nil
		},
	}
	mockTokens := &MockTokenRepository{}

	svc := NewService(mockRepo, mockTokens)

	_, _, _, err := svc.CreateStaffUser(context.Background(), uuid.New(), actorRoles, CreateStaffUserRequest{
		Email:        "existing@example.com",
		Password:     "password123",
		FullNameEN:   "Staff",
		StaffProfile: UpdateStaffProfileRequest{},
	})
	if !errors.Is(err, ErrEmailExists) {
		t.Errorf("CreateStaffUser() error = %v, want ErrEmailExists", err)
	}
}

func TestCreateStaffUser_ScopeIDRequired(t *testing.T) {
	actorRoles := []auth.RoleClaim{{Permission: "admin", ScopeType: "university"}}
	mockRepo := &MockUserRepository{}
	mockTokens := &MockTokenRepository{}

	svc := NewService(mockRepo, mockTokens)

	_, _, _, err := svc.CreateStaffUser(context.Background(), uuid.New(), actorRoles, CreateStaffUserRequest{
		Email:        "staff@example.com",
		Password:     "password123",
		FullNameEN:   "Staff",
		StaffProfile: UpdateStaffProfileRequest{},
		Role: &CreateRoleRequest{
			Permission: "admin",
			ScopeType:  "college",
			ScopeID:    nil,
		},
	})
	if !errors.Is(err, ErrScopeIDRequired) {
		t.Errorf("CreateStaffUser() error = %v, want ErrScopeIDRequired", err)
	}
}

func TestCreateStaffUser_ScopeIDNotAllowed(t *testing.T) {
	scopeID := uuid.New()
	actorRoles := []auth.RoleClaim{{Permission: "admin", ScopeType: "university"}}

	mockRepo := &MockUserRepository{}
	mockTokens := &MockTokenRepository{}

	svc := NewService(mockRepo, mockTokens)

	_, _, _, err := svc.CreateStaffUser(context.Background(), uuid.New(), actorRoles, CreateStaffUserRequest{
		Email:        "staff@example.com",
		Password:     "password123",
		FullNameEN:   "Staff",
		StaffProfile: UpdateStaffProfileRequest{},
		Role: &CreateRoleRequest{
			Permission: "admin",
			ScopeType:  "university",
			ScopeID:    &scopeID,
		},
	})
	if !errors.Is(err, ErrScopeIDNotAllowed) {
		t.Errorf("CreateStaffUser() error = %v, want ErrScopeIDNotAllowed", err)
	}
}

func TestCreateStaffUser_CannotManageHigherRole(t *testing.T) {
	actorRoles := []auth.RoleClaim{{Permission: "admin", ScopeType: "university"}}
	mockRepo := &MockUserRepository{}
	mockTokens := &MockTokenRepository{}

	svc := NewService(mockRepo, mockTokens)

	_, _, _, err := svc.CreateStaffUser(context.Background(), uuid.New(), actorRoles, CreateStaffUserRequest{
		Email:        "staff@example.com",
		Password:     "password123",
		FullNameEN:   "Staff",
		StaffProfile: UpdateStaffProfileRequest{},
		Role: &CreateRoleRequest{
			Permission: "super_admin",
			ScopeType:  "university",
		},
	})
	if !errors.Is(err, ErrCannotManageHigherRole) {
		t.Errorf("CreateStaffUser() error = %v, want ErrCannotManageHigherRole", err)
	}
}
