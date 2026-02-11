// Package user handles user profile and session management.
package user

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/ranjdotdev/e-campus-server/internal/auth"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidPassword  = errors.New("invalid password")
	ErrSameEmail        = errors.New("new email is the same as current")
	ErrSamePassword     = errors.New("new password is the same as current")
	ErrSessionNotFound  = errors.New("session not found")
	ErrCannotDeactivate = errors.New("cannot deactivate user")
	ErrScopeIDRequired   = errors.New("scope_id required for non-university scope")
	ErrScopeIDNotAllowed = errors.New("scope_id not allowed for university scope")
)

type UserRepository interface {
	GetUser(ctx context.Context, id uuid.UUID) (*User, error)
	Update(ctx context.Context, user *User) error
	UpdateEmail(ctx context.Context, id uuid.UUID, email string) error
	EmailExists(ctx context.Context, email string) (bool, error)
	GetPasswordHash(ctx context.Context, id uuid.UUID) (string, error)
	SetPassword(ctx context.Context, userID uuid.UUID, passwordHash string) error
	List(ctx context.Context, limit, offset int) ([]User, int, error)
	Deactivate(ctx context.Context, id uuid.UUID) error
	GetRoles(ctx context.Context, userID uuid.UUID) ([]Role, error)
	GetStaffProfile(ctx context.Context, userID uuid.UUID) (*StaffProfile, error)
	CreateStaffProfile(ctx context.Context, profile *StaffProfile) error
	UpdateStaffProfile(ctx context.Context, profile *StaffProfile) error
	CreateStaffUserTx(ctx context.Context, user *User, profile *StaffProfile, role *Role) error
}

type Service struct {
	repo   UserRepository
	tokens auth.TokenRepository
}

func NewService(repo UserRepository, tokens auth.TokenRepository) *Service {
	return &Service{repo: repo, tokens: tokens}
}

func (s *Service) GetProfile(ctx context.Context, userID uuid.UUID) (*User, error) {
	return s.repo.GetUser(ctx, userID)
}

func (s *Service) UpdateProfile(ctx context.Context, userID uuid.UUID, req UpdateProfileRequest) (*User, error) {
	user, err := s.repo.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	if req.FullNameEN != nil {
		user.FullNameEN = *req.FullNameEN
	}
	if req.FullNameKU != nil {
		user.FullNameKU = req.FullNameKU
	}
	if req.AvatarURL != nil {
		user.AvatarURL = req.AvatarURL
	}
	if req.Phone != nil {
		user.Phone = req.Phone
	}

	if err := s.repo.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *Service) UpdateEmail(ctx context.Context, userID uuid.UUID, req UpdateEmailRequest) error {
	user, err := s.repo.GetUser(ctx, userID)
	if err != nil {
		return err
	}

	if user.Email == req.Email {
		return ErrSameEmail
	}

	hash, err := s.repo.GetPasswordHash(ctx, userID)
	if err != nil {
		return err
	}

	if !checkPassword(req.Password, hash) {
		return ErrInvalidPassword
	}

	exists, err := s.repo.EmailExists(ctx, req.Email)
	if err != nil {
		return err
	}
	if exists {
		return ErrEmailExists
	}

	return s.repo.UpdateEmail(ctx, userID, req.Email)
}

func (s *Service) ListUsers(ctx context.Context, limit, offset int) ([]User, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	return s.repo.List(ctx, limit, offset)
}

func (s *Service) GetUserByID(ctx context.Context, userID uuid.UUID) (*User, error) {
	return s.repo.GetUser(ctx, userID)
}

func (s *Service) DeactivateUser(ctx context.Context, userID uuid.UUID) error {
	return s.repo.Deactivate(ctx, userID)
}

func (s *Service) GetRoles(ctx context.Context, userID uuid.UUID) ([]Role, error) {
	return s.repo.GetRoles(ctx, userID)
}

func (s *Service) GetSessions(ctx context.Context, userID uuid.UUID) ([]Session, error) {
	tokens, err := s.tokens.GetUserSessions(ctx, userID)
	if err != nil {
		return nil, err
	}

	sessions := make([]Session, len(tokens))
	for i, t := range tokens {
		sessions[i] = Session{
			ID:        t.ID,
			Device:    t.Device,
			IPAddress: t.IPAddress,
			CreatedAt: t.CreatedAt,
			ExpiresAt: t.ExpiresAt,
			UsedAt:    t.UsedAt,
		}
	}
	return sessions, nil
}

func (s *Service) RevokeSession(ctx context.Context, userID, sessionID uuid.UUID) error {
	tokens, err := s.tokens.GetUserSessions(ctx, userID)
	if err != nil {
		return err
	}

	for _, token := range tokens {
		if token.ID == sessionID {
			return s.tokens.DeleteToken(ctx, token.TokenHash)
		}
	}

	return ErrSessionNotFound
}

func (s *Service) GetStaffProfile(ctx context.Context, userID uuid.UUID) (*StaffProfile, error) {
	return s.repo.GetStaffProfile(ctx, userID)
}

func (s *Service) CreateStaffProfile(ctx context.Context, userID uuid.UUID, req UpdateStaffProfileRequest) (*StaffProfile, error) {
	if _, err := s.repo.GetUser(ctx, userID); err != nil {
		return nil, err
	}

	profile := &StaffProfile{
		UserID:         userID,
		HighestDegree:  req.HighestDegree,
		FieldOfStudy:   req.FieldOfStudy,
		YearsOfService: derefInt(req.YearsOfService, 0),
		Salary:         req.SalaryString(),
		SalaryCurrency: req.SalaryCurrency,
	}

	if err := s.repo.CreateStaffProfile(ctx, profile); err != nil {
		return nil, err
	}

	return profile, nil
}

func (s *Service) UpdateStaffProfile(ctx context.Context, userID uuid.UUID, req UpdateStaffProfileRequest) (*StaffProfile, error) {
	profile, err := s.repo.GetStaffProfile(ctx, userID)
	if err != nil {
		return nil, err
	}

	if req.HighestDegree != nil {
		profile.HighestDegree = req.HighestDegree
	}
	if req.FieldOfStudy != nil {
		profile.FieldOfStudy = req.FieldOfStudy
	}
	if req.YearsOfService != nil {
		profile.YearsOfService = *req.YearsOfService
	}
	if req.Salary != nil {
		profile.Salary = req.SalaryString()
	}
	if req.SalaryCurrency != nil {
		profile.SalaryCurrency = req.SalaryCurrency
	}

	if err := s.repo.UpdateStaffProfile(ctx, profile); err != nil {
		return nil, err
	}

	return profile, nil
}

func (s *Service) CreateStaffUser(ctx context.Context, adminID uuid.UUID, req CreateStaffUserRequest) (*User, *StaffProfile, *Role, error) {
	if req.Role != nil {
		if req.Role.ScopeType == "university" && req.Role.ScopeID != nil {
			return nil, nil, nil, ErrScopeIDNotAllowed
		}
		if req.Role.ScopeType != "university" && req.Role.ScopeID == nil {
			return nil, nil, nil, ErrScopeIDRequired
		}
	}

	exists, err := s.repo.EmailExists(ctx, req.Email)
	if err != nil {
		return nil, nil, nil, err
	}
	if exists {
		return nil, nil, nil, ErrEmailExists
	}

	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, nil, nil, err
	}

	user := &User{
		Email:        req.Email,
		PasswordHash: passwordHash,
		FullNameEN:   req.FullNameEN,
		FullNameKU:   req.FullNameKU,
	}

	profile := &StaffProfile{
		HighestDegree:  req.StaffProfile.HighestDegree,
		FieldOfStudy:   req.StaffProfile.FieldOfStudy,
		YearsOfService: derefInt(req.StaffProfile.YearsOfService, 0),
		Salary:         req.StaffProfile.SalaryString(),
		SalaryCurrency: req.StaffProfile.SalaryCurrency,
	}

	var role *Role
	if req.Role != nil {
		role = &Role{
			Title:      req.Role.Title,
			Permission: req.Role.Permission,
			ScopeType:  req.Role.ScopeType,
			ScopeID:    req.Role.ScopeID,
			AssignedBy: &adminID,
		}
	}

	if err := s.repo.CreateStaffUserTx(ctx, user, profile, role); err != nil {
		return nil, nil, nil, err
	}

	return user, profile, role, nil
}

func (s *Service) AdminSetPassword(ctx context.Context, userID uuid.UUID, password string) error {
	if _, err := s.repo.GetUser(ctx, userID); err != nil {
		return err
	}

	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}

	return s.repo.SetPassword(ctx, userID, passwordHash)
}

func (s *Service) ChangePassword(ctx context.Context, userID uuid.UUID, req ChangePasswordRequest) error {
	hash, err := s.repo.GetPasswordHash(ctx, userID)
	if err != nil {
		return err
	}

	if !auth.CheckPassword(req.CurrentPassword, hash) {
		return ErrInvalidPassword
	}

	if req.CurrentPassword == req.NewPassword {
		return ErrSamePassword
	}

	passwordHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		return err
	}

	return s.repo.SetPassword(ctx, userID, passwordHash)
}

func checkPassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func derefInt(p *int, defaultVal int) int {
	if p == nil {
		return defaultVal
	}
	return *p
}
