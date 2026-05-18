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

// ── Mocks ────────────────────────────────────────────────────────────────────

type MockRoleManager struct{}

func (m *MockRoleManager) CanManageRole(ctx context.Context, actor, target *auth.RoleClaim) bool {
	return true
}

type MockUserRepository struct {
	GetUserFunc            func(ctx context.Context, id uuid.UUID) (*User, error)
	UpdateFunc             func(ctx context.Context, user *User) error
	UpdateEmailFunc        func(ctx context.Context, id uuid.UUID, email string) error
	EmailExistsFunc        func(ctx context.Context, email string) (bool, error)
	GetPasswordHashFunc    func(ctx context.Context, id uuid.UUID) (string, error)
	SetPasswordFunc        func(ctx context.Context, userID uuid.UUID, passwordHash string) error
	ListFunc               func(ctx context.Context, params pagination.PageParams, filters UserFilters) ([]User, bool, error)
	DeactivateFunc         func(ctx context.Context, id uuid.UUID) error
	GetRoleFunc            func(ctx context.Context, userID uuid.UUID) (*Role, error)
	CreateRoleFunc         func(ctx context.Context, role *Role) error
	UpdateRoleFunc         func(ctx context.Context, role *Role) error
	DeleteRoleFunc         func(ctx context.Context, userID uuid.UUID) error
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
func (m *MockUserRepository) GetRole(ctx context.Context, userID uuid.UUID) (*Role, error) {
	if m.GetRoleFunc != nil {
		return m.GetRoleFunc(ctx, userID)
	}
	return nil, nil
}
func (m *MockUserRepository) CreateRole(ctx context.Context, role *Role) error {
	if m.CreateRoleFunc != nil {
		return m.CreateRoleFunc(ctx, role)
	}
	role.ID = uuid.New()
	return nil
}
func (m *MockUserRepository) UpdateRole(ctx context.Context, role *Role) error {
	if m.UpdateRoleFunc != nil {
		return m.UpdateRoleFunc(ctx, role)
	}
	return nil
}
func (m *MockUserRepository) DeleteRole(ctx context.Context, userID uuid.UUID) error {
	if m.DeleteRoleFunc != nil {
		return m.DeleteRoleFunc(ctx, userID)
	}
	return nil
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
func (m *MockTokenRepository) MarkTokenUsed(ctx context.Context, hash string) (bool, error) {
	return false, nil
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

// ── GetProfile ───────────────────────────────────────────────────────────────

func TestGetProfile_Success(t *testing.T) {
	userID := uuid.New()
	repo := &MockUserRepository{
		GetUserFunc: func(ctx context.Context, id uuid.UUID) (*User, error) {
			return &User{ID: userID, Email: "test@example.com", FullNameEN: "Test User", IsActive: true}, nil
		},
	}
	service := NewService(repo, &MockTokenRepository{}, nil, &MockRoleManager{}, nil, nil, nil)

	user, err := service.GetProfile(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetProfile() error = %v", err)
	}
	if user.ID != userID {
		t.Errorf("user.ID = %v, want %v", user.ID, userID)
	}
}

func TestGetProfile_NotFound(t *testing.T) {
	repo := &MockUserRepository{
		GetUserFunc: func(ctx context.Context, id uuid.UUID) (*User, error) { return nil, ErrUserNotFound },
	}
	service := NewService(repo, &MockTokenRepository{}, nil, &MockRoleManager{}, nil, nil, nil)

	_, err := service.GetProfile(context.Background(), uuid.New())
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("GetProfile() error = %v, want ErrUserNotFound", err)
	}
}

// ── UpdateProfile ────────────────────────────────────────────────────────────

func TestUpdateProfile_Success(t *testing.T) {
	userID := uuid.New()
	newName := "Updated Name"
	repo := &MockUserRepository{
		GetUserFunc: func(ctx context.Context, id uuid.UUID) (*User, error) {
			return &User{ID: userID, Email: "test@example.com", FullNameEN: "Original Name"}, nil
		},
		UpdateFunc: func(ctx context.Context, user *User) error {
			if user.FullNameEN != newName {
				t.Errorf("user.FullNameEN = %v, want %v", user.FullNameEN, newName)
			}
			return nil
		},
	}
	service := NewService(repo, &MockTokenRepository{}, nil, &MockRoleManager{}, nil, nil, nil)

	user, err := service.UpdateProfile(context.Background(), userID, UpdateProfileRequest{FullNameEN: &newName})
	if err != nil {
		t.Fatalf("UpdateProfile() error = %v", err)
	}
	if user.FullNameEN != newName {
		t.Errorf("user.FullNameEN = %v, want %v", user.FullNameEN, newName)
	}
}

// ── UpdateEmail ──────────────────────────────────────────────────────────────

func TestUpdateEmail_Success(t *testing.T) {
	userID := uuid.New()
	passwordHash, _ := auth.HashPassword("Password123")
	repo := &MockUserRepository{
		GetUserFunc: func(ctx context.Context, id uuid.UUID) (*User, error) {
			return &User{ID: userID, Email: "old@example.com"}, nil
		},
		GetPasswordHashFunc: func(ctx context.Context, id uuid.UUID) (string, error) { return passwordHash, nil },
		EmailExistsFunc:     func(ctx context.Context, email string) (bool, error) { return false, nil },
		UpdateEmailFunc:     func(ctx context.Context, id uuid.UUID, email string) error { return nil },
	}
	service := NewService(repo, &MockTokenRepository{}, nil, &MockRoleManager{}, nil, nil, nil)

	err := service.UpdateEmail(context.Background(), userID, UpdateEmailRequest{Email: "new@example.com", Password: "Password123"})
	if err != nil {
		t.Fatalf("UpdateEmail() error = %v", err)
	}
}

func TestUpdateEmail_SameEmail(t *testing.T) {
	userID := uuid.New()
	repo := &MockUserRepository{
		GetUserFunc: func(ctx context.Context, id uuid.UUID) (*User, error) {
			return &User{ID: userID, Email: "same@example.com"}, nil
		},
	}
	service := NewService(repo, &MockTokenRepository{}, nil, &MockRoleManager{}, nil, nil, nil)

	err := service.UpdateEmail(context.Background(), userID, UpdateEmailRequest{Email: "same@example.com", Password: "Password123"})
	if !errors.Is(err, ErrSameEmail) {
		t.Errorf("UpdateEmail() error = %v, want ErrSameEmail", err)
	}
}

func TestUpdateEmail_InvalidPassword(t *testing.T) {
	userID := uuid.New()
	passwordHash, _ := auth.HashPassword("correctpassword")
	repo := &MockUserRepository{
		GetUserFunc: func(ctx context.Context, id uuid.UUID) (*User, error) {
			return &User{ID: userID, Email: "old@example.com"}, nil
		},
		GetPasswordHashFunc: func(ctx context.Context, id uuid.UUID) (string, error) { return passwordHash, nil },
	}
	service := NewService(repo, &MockTokenRepository{}, nil, &MockRoleManager{}, nil, nil, nil)

	err := service.UpdateEmail(context.Background(), userID, UpdateEmailRequest{Email: "new@example.com", Password: "wrongpassword"})
	if !errors.Is(err, ErrInvalidPassword) {
		t.Errorf("UpdateEmail() error = %v, want ErrInvalidPassword", err)
	}
}

func TestUpdateEmail_EmailExists(t *testing.T) {
	userID := uuid.New()
	passwordHash, _ := auth.HashPassword("Password123")
	repo := &MockUserRepository{
		GetUserFunc: func(ctx context.Context, id uuid.UUID) (*User, error) {
			return &User{ID: userID, Email: "old@example.com"}, nil
		},
		GetPasswordHashFunc: func(ctx context.Context, id uuid.UUID) (string, error) { return passwordHash, nil },
		EmailExistsFunc:     func(ctx context.Context, email string) (bool, error) { return true, nil },
	}
	service := NewService(repo, &MockTokenRepository{}, nil, &MockRoleManager{}, nil, nil, nil)

	err := service.UpdateEmail(context.Background(), userID, UpdateEmailRequest{Email: "taken@example.com", Password: "Password123"})
	if !errors.Is(err, ErrEmailExists) {
		t.Errorf("UpdateEmail() error = %v, want ErrEmailExists", err)
	}
}

// ── Sessions ─────────────────────────────────────────────────────────────────

func TestGetSessions_Success(t *testing.T) {
	userID := uuid.New()
	sessionID := uuid.New()
	device := "Test Device"
	ip := "127.0.0.1"

	tokens := &MockTokenRepository{
		GetUserSessionsFunc: func(ctx context.Context, uid uuid.UUID) ([]auth.RefreshToken, error) {
			return []auth.RefreshToken{
				{ID: sessionID, UserID: userID, Device: &device, IPAddress: &ip, CreatedAt: time.Now(), ExpiresAt: time.Now().Add(time.Hour)},
			}, nil
		},
	}
	service := NewService(&MockUserRepository{}, tokens, nil, &MockRoleManager{}, nil, nil, nil)

	sessions, err := service.GetSessions(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetSessions() error = %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("len(sessions) = %v, want 1", len(sessions))
	}
	if sessions[0].ID != sessionID {
		t.Errorf("sessions[0].ID = %v, want %v", sessions[0].ID, sessionID)
	}
}

func TestRevokeSession_Success(t *testing.T) {
	userID := uuid.New()
	sessionID := uuid.New()
	tokenHash := "test-hash"
	called := false

	tokens := &MockTokenRepository{
		GetUserSessionsFunc: func(ctx context.Context, uid uuid.UUID) ([]auth.RefreshToken, error) {
			return []auth.RefreshToken{{ID: sessionID, UserID: userID, TokenHash: tokenHash}}, nil
		},
		DeleteTokenFunc: func(ctx context.Context, hash string) error {
			if hash != tokenHash {
				t.Errorf("DeleteToken called with %v, want %v", hash, tokenHash)
			}
			called = true
			return nil
		},
	}
	service := NewService(&MockUserRepository{}, tokens, nil, &MockRoleManager{}, nil, nil, nil)

	if err := service.RevokeSession(context.Background(), userID, sessionID); err != nil {
		t.Fatalf("RevokeSession() error = %v", err)
	}
	if !called {
		t.Error("DeleteToken was not called")
	}
}

func TestRevokeSession_NotFound(t *testing.T) {
	tokens := &MockTokenRepository{
		GetUserSessionsFunc: func(ctx context.Context, uid uuid.UUID) ([]auth.RefreshToken, error) {
			return []auth.RefreshToken{}, nil
		},
	}
	service := NewService(&MockUserRepository{}, tokens, nil, &MockRoleManager{}, nil, nil, nil)

	err := service.RevokeSession(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("RevokeSession() error = %v, want ErrSessionNotFound", err)
	}
}

// ── StaffProfile ─────────────────────────────────────────────────────────────

func TestCreateStaffProfile_Success(t *testing.T) {
	userID := uuid.New()
	degree := "phd"
	repo := &MockUserRepository{
		GetUserFunc:         func(ctx context.Context, id uuid.UUID) (*User, error) { return &User{ID: userID}, nil },
		GetStaffProfileFunc: func(ctx context.Context, uid uuid.UUID) (*StaffProfile, error) { return nil, ErrStaffProfileNotFound },
		CreateStaffProfileFunc: func(ctx context.Context, profile *StaffProfile) error {
			profile.ID = uuid.New()
			return nil
		},
	}
	service := NewService(repo, &MockTokenRepository{}, nil, &MockRoleManager{}, nil, nil, nil)

	profile, err := service.CreateStaffProfile(context.Background(), userID, UpdateStaffProfileRequest{HighestDegree: &degree})
	if err != nil {
		t.Fatalf("CreateStaffProfile() error = %v", err)
	}
	if profile.HighestDegree == nil || *profile.HighestDegree != degree {
		t.Errorf("profile.HighestDegree = %v, want %v", profile.HighestDegree, degree)
	}
}

func TestCreateStaffProfile_AlreadyExists(t *testing.T) {
	userID := uuid.New()
	repo := &MockUserRepository{
		GetUserFunc: func(ctx context.Context, id uuid.UUID) (*User, error) { return &User{ID: userID}, nil },
		GetStaffProfileFunc: func(ctx context.Context, uid uuid.UUID) (*StaffProfile, error) {
			return &StaffProfile{ID: uuid.New()}, nil
		},
	}
	service := NewService(repo, &MockTokenRepository{}, nil, &MockRoleManager{}, nil, nil, nil)

	_, err := service.CreateStaffProfile(context.Background(), userID, UpdateStaffProfileRequest{})
	if !errors.Is(err, ErrStaffProfileExists) {
		t.Errorf("CreateStaffProfile() error = %v, want ErrStaffProfileExists", err)
	}
}

// ── ChangePassword ────────────────────────────────────────────────────────────

func TestChangePassword_Success(t *testing.T) {
	userID := uuid.New()
	currentHash, _ := auth.HashPassword("Current1pass")
	tokensCalled := false

	repo := &MockUserRepository{
		GetPasswordHashFunc: func(ctx context.Context, id uuid.UUID) (string, error) { return currentHash, nil },
		SetPasswordFunc:     func(ctx context.Context, uid uuid.UUID, hash string) error { return nil },
	}
	tokens := &MockTokenRepository{
		DeleteUserTokensFunc: func(ctx context.Context, uid uuid.UUID) error { tokensCalled = true; return nil },
	}
	service := NewService(repo, tokens, nil, &MockRoleManager{}, nil, nil, nil)

	err := service.ChangePassword(context.Background(), userID, ChangePasswordRequest{
		CurrentPassword: "Current1pass",
		NewPassword:     "NewPass1word",
	})
	if err != nil {
		t.Fatalf("ChangePassword() error = %v", err)
	}
	if !tokensCalled {
		t.Error("DeleteUserTokens was not called")
	}
}

func TestChangePassword_InvalidCurrentPassword(t *testing.T) {
	currentHash, _ := auth.HashPassword("correctpassword")
	repo := &MockUserRepository{
		GetPasswordHashFunc: func(ctx context.Context, id uuid.UUID) (string, error) { return currentHash, nil },
	}
	service := NewService(repo, &MockTokenRepository{}, nil, &MockRoleManager{}, nil, nil, nil)

	err := service.ChangePassword(context.Background(), uuid.New(), ChangePasswordRequest{
		CurrentPassword: "wrongpassword",
		NewPassword:     "NewPass1word",
	})
	if !errors.Is(err, ErrInvalidPassword) {
		t.Errorf("ChangePassword() error = %v, want ErrInvalidPassword", err)
	}
}

func TestChangePassword_SamePassword(t *testing.T) {
	currentHash, _ := auth.HashPassword("Same1password")
	repo := &MockUserRepository{
		GetPasswordHashFunc: func(ctx context.Context, id uuid.UUID) (string, error) { return currentHash, nil },
	}
	service := NewService(repo, &MockTokenRepository{}, nil, &MockRoleManager{}, nil, nil, nil)

	err := service.ChangePassword(context.Background(), uuid.New(), ChangePasswordRequest{
		CurrentPassword: "Same1password",
		NewPassword:     "Same1password",
	})
	if !errors.Is(err, ErrSamePassword) {
		t.Errorf("ChangePassword() error = %v, want ErrSamePassword", err)
	}
}

func TestChangePassword_WeakNewPassword(t *testing.T) {
	currentHash, _ := auth.HashPassword("Current1pass")
	repo := &MockUserRepository{
		GetPasswordHashFunc: func(ctx context.Context, id uuid.UUID) (string, error) { return currentHash, nil },
	}
	service := NewService(repo, &MockTokenRepository{}, nil, &MockRoleManager{}, nil, nil, nil)

	err := service.ChangePassword(context.Background(), uuid.New(), ChangePasswordRequest{
		CurrentPassword: "Current1pass",
		NewPassword:     "weak", // too short, no digit, no upper
	})
	if !errors.Is(err, auth.ErrPasswordTooShort) && !errors.Is(err, auth.ErrPasswordTooWeak) {
		t.Errorf("ChangePassword() error = %v, want password strength error", err)
	}
}

// ── AdminSetPassword ──────────────────────────────────────────────────────────

func TestAdminSetPassword_Success(t *testing.T) {
	userID := uuid.New()
	tokensCalled := false

	repo := &MockUserRepository{
		GetUserFunc:     func(ctx context.Context, id uuid.UUID) (*User, error) { return &User{ID: userID}, nil },
		SetPasswordFunc: func(ctx context.Context, uid uuid.UUID, hash string) error { return nil },
	}
	tokens := &MockTokenRepository{
		DeleteUserTokensFunc: func(ctx context.Context, uid uuid.UUID) error { tokensCalled = true; return nil },
	}
	service := NewService(repo, tokens, nil, &MockRoleManager{}, nil, nil, nil)

	err := service.AdminSetPassword(context.Background(), userID, "StrongPass1")
	if err != nil {
		t.Fatalf("AdminSetPassword() error = %v", err)
	}
	if !tokensCalled {
		t.Error("DeleteUserTokens was not called")
	}
}

func TestAdminSetPassword_WeakPassword(t *testing.T) {
	repo := &MockUserRepository{}
	service := NewService(repo, &MockTokenRepository{}, nil, &MockRoleManager{}, nil, nil, nil)

	err := service.AdminSetPassword(context.Background(), uuid.New(), "weak")
	if !errors.Is(err, auth.ErrPasswordTooShort) && !errors.Is(err, auth.ErrPasswordTooWeak) {
		t.Errorf("AdminSetPassword() error = %v, want password strength error", err)
	}
}

func TestAdminSetPassword_UserNotFound(t *testing.T) {
	repo := &MockUserRepository{
		GetUserFunc: func(ctx context.Context, id uuid.UUID) (*User, error) { return nil, ErrUserNotFound },
	}
	service := NewService(repo, &MockTokenRepository{}, nil, &MockRoleManager{}, nil, nil, nil)

	err := service.AdminSetPassword(context.Background(), uuid.New(), "StrongPass1")
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("AdminSetPassword() error = %v, want ErrUserNotFound", err)
	}
}

// ── CreateStaffUser ───────────────────────────────────────────────────────────

func TestCreateStaffUser_Success(t *testing.T) {
	adminID := uuid.New()
	actorRole := &auth.RoleClaim{Level: "admin", ScopeType: "university"}

	repo := &MockUserRepository{
		EmailExistsFunc: func(ctx context.Context, email string) (bool, error) { return false, nil },
		CreateStaffUserTxFunc: func(ctx context.Context, user *User, profile *StaffProfile, role *Role) error {
			user.ID = uuid.New()
			profile.ID = uuid.New()
			return nil
		},
	}
	service := NewService(repo, &MockTokenRepository{}, nil, &MockRoleManager{}, nil, nil, nil)

	user, profile, _, err := service.CreateStaffUser(context.Background(), adminID, actorRole, CreateStaffUserRequest{
		Email:        "new@example.com",
		Password:     "StrongPass1",
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

func TestCreateStaffUser_WeakPassword(t *testing.T) {
	actorRole := &auth.RoleClaim{Level: "admin", ScopeType: "university"}
	service := NewService(&MockUserRepository{}, &MockTokenRepository{}, nil, &MockRoleManager{}, nil, nil, nil)

	_, _, _, err := service.CreateStaffUser(context.Background(), uuid.New(), actorRole, CreateStaffUserRequest{
		Email:        "staff@example.com",
		Password:     "weak",
		FullNameEN:   "Staff",
		StaffProfile: UpdateStaffProfileRequest{},
	})
	if !errors.Is(err, auth.ErrPasswordTooShort) && !errors.Is(err, auth.ErrPasswordTooWeak) {
		t.Errorf("CreateStaffUser() error = %v, want password strength error", err)
	}
}

func TestCreateStaffUser_EmailExists(t *testing.T) {
	actorRole := &auth.RoleClaim{Level: "admin", ScopeType: "university"}
	repo := &MockUserRepository{
		EmailExistsFunc: func(ctx context.Context, email string) (bool, error) { return true, nil },
	}
	service := NewService(repo, &MockTokenRepository{}, nil, &MockRoleManager{}, nil, nil, nil)

	_, _, _, err := service.CreateStaffUser(context.Background(), uuid.New(), actorRole, CreateStaffUserRequest{
		Email:        "existing@example.com",
		Password:     "StrongPass1",
		FullNameEN:   "Staff",
		StaffProfile: UpdateStaffProfileRequest{},
	})
	if !errors.Is(err, ErrEmailExists) {
		t.Errorf("CreateStaffUser() error = %v, want ErrEmailExists", err)
	}
}

func TestCreateStaffUser_ScopeIDRequired(t *testing.T) {
	actorRole := &auth.RoleClaim{Level: "admin", ScopeType: "university"}
	service := NewService(&MockUserRepository{}, &MockTokenRepository{}, nil, &MockRoleManager{}, nil, nil, nil)

	_, _, _, err := service.CreateStaffUser(context.Background(), uuid.New(), actorRole, CreateStaffUserRequest{
		Email:        "staff@example.com",
		Password:     "StrongPass1",
		FullNameEN:   "Staff",
		StaffProfile: UpdateStaffProfileRequest{},
		Role:         &CreateRoleRequest{Level: "admin", ScopeType: "college", ScopeID: nil},
	})
	if !errors.Is(err, ErrScopeIDRequired) {
		t.Errorf("CreateStaffUser() error = %v, want ErrScopeIDRequired", err)
	}
}

func TestCreateStaffUser_ScopeIDNotAllowed(t *testing.T) {
	scopeID := uuid.New()
	actorRole := &auth.RoleClaim{Level: "admin", ScopeType: "platform"}
	service := NewService(&MockUserRepository{}, &MockTokenRepository{}, nil, &MockRoleManager{}, nil, nil, nil)

	_, _, _, err := service.CreateStaffUser(context.Background(), uuid.New(), actorRole, CreateStaffUserRequest{
		Email:        "staff@example.com",
		Password:     "StrongPass1",
		FullNameEN:   "Staff",
		StaffProfile: UpdateStaffProfileRequest{},
		Role:         &CreateRoleRequest{Level: "admin", ScopeType: "university", ScopeID: &scopeID},
	})
	if !errors.Is(err, ErrScopeIDNotAllowed) {
		t.Errorf("CreateStaffUser() error = %v, want ErrScopeIDNotAllowed", err)
	}
}

func TestCreateStaffUser_CannotManageHigherRole(t *testing.T) {
	actorRole := &auth.RoleClaim{Level: "admin", ScopeType: "university"}
	service := NewService(&MockUserRepository{}, &MockTokenRepository{}, nil, &MockRoleManager{}, nil, nil, nil)

	_, _, _, err := service.CreateStaffUser(context.Background(), uuid.New(), actorRole, CreateStaffUserRequest{
		Email:        "staff@example.com",
		Password:     "StrongPass1",
		FullNameEN:   "Staff",
		StaffProfile: UpdateStaffProfileRequest{},
		Role:         &CreateRoleRequest{Level: "super_admin", ScopeType: "university"},
	})
	if !errors.Is(err, ErrCannotManageHigherRole) {
		t.Errorf("CreateStaffUser() error = %v, want ErrCannotManageHigherRole", err)
	}
}
