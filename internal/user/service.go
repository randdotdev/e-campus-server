package user

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/ranjdotdev/e-campus-server/internal/auth"
	"github.com/ranjdotdev/e-campus-server/internal/authz"
	"github.com/ranjdotdev/e-campus-server/internal/pagination"
)

type RoleManager interface {
	CanManageRole(ctx context.Context, actor, target *auth.RoleClaim) bool
}

type UserRepository interface {
	GetUser(ctx context.Context, id uuid.UUID) (*User, error)
	Update(ctx context.Context, user *User) error
	UpdateEmail(ctx context.Context, id uuid.UUID, email string) error
	EmailExists(ctx context.Context, email string) (bool, error)
	GetPasswordHash(ctx context.Context, id uuid.UUID) (string, error)
	SetPassword(ctx context.Context, userID uuid.UUID, passwordHash string) error
	List(ctx context.Context, params pagination.PageParams, filters UserFilters) ([]User, bool, error)
	Deactivate(ctx context.Context, id uuid.UUID) error
	GetRole(ctx context.Context, userID uuid.UUID) (*Role, error)
	CreateRole(ctx context.Context, role *Role) error
	UpdateRole(ctx context.Context, role *Role) error
	DeleteRole(ctx context.Context, userID uuid.UUID) error
	GetStaffProfile(ctx context.Context, userID uuid.UUID) (*StaffProfile, error)
	CreateStaffProfile(ctx context.Context, profile *StaffProfile) error
	UpdateStaffProfile(ctx context.Context, profile *StaffProfile) error
	ScopeExists(ctx context.Context, scopeType string, scopeID uuid.UUID) (bool, error)
	CreateStaffUserTx(ctx context.Context, user *User, profile *StaffProfile, role *Role) error
}

type Notifier interface {
	Send(ctx context.Context, userID uuid.UUID, notifType, title string, body *string, data map[string]any) error
}

type Service struct {
	repo     UserRepository
	tokens   auth.TokenRepository
	notifier Notifier
	roles    RoleManager
}

func NewService(repo UserRepository, tokens auth.TokenRepository, notifier Notifier, roles RoleManager) *Service {
	return &Service{repo: repo, tokens: tokens, notifier: notifier, roles: roles}
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
	if req.FullNameLocal != nil {
		if *req.FullNameLocal == "" {
			user.FullNameLocal = nil
		} else {
			user.FullNameLocal = req.FullNameLocal
		}
	}
	if req.AvatarURL != nil {
		if *req.AvatarURL == "" {
			user.AvatarURL = nil
		} else {
			user.AvatarURL = req.AvatarURL
		}
	}
	if req.Phone != nil {
		if *req.Phone == "" {
			user.Phone = nil
		} else {
			user.Phone = req.Phone
		}
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

func (s *Service) ListUsers(ctx context.Context, params pagination.PageParams, filters UserFilters) ([]User, bool, error) {
	return s.repo.List(ctx, params, filters)
}

func (s *Service) GetUserByID(ctx context.Context, userID uuid.UUID) (*User, error) {
	return s.repo.GetUser(ctx, userID)
}

func (s *Service) DeactivateUser(ctx context.Context, userID uuid.UUID) error {
	if err := s.repo.Deactivate(ctx, userID); err != nil {
		return err
	}
	return s.tokens.DeleteUserTokens(ctx, userID)
}

func (s *Service) GetRole(ctx context.Context, userID uuid.UUID) (*Role, error) {
	return s.repo.GetRole(ctx, userID)
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

func (s *Service) RevokeOtherSessions(ctx context.Context, userID, keepSessionID uuid.UUID) error {
	tokens, err := s.tokens.GetUserSessions(ctx, userID)
	if err != nil {
		return err
	}

	for _, token := range tokens {
		if token.ID != keepSessionID {
			if err := s.tokens.DeleteToken(ctx, token.TokenHash); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) GetStaffProfile(ctx context.Context, userID uuid.UUID) (*StaffProfile, error) {
	return s.repo.GetStaffProfile(ctx, userID)
}

func (s *Service) CreateStaffProfile(ctx context.Context, userID uuid.UUID, req UpdateStaffProfileRequest) (*StaffProfile, error) {
	if _, err := s.repo.GetUser(ctx, userID); err != nil {
		return nil, err
	}

	if _, err := s.repo.GetStaffProfile(ctx, userID); err == nil {
		return nil, ErrStaffProfileExists
	} else if !errors.Is(err, ErrStaffProfileNotFound) {
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

func (s *Service) CreateStaffUser(ctx context.Context, adminID uuid.UUID, actorRole *auth.RoleClaim, req CreateStaffUserRequest) (*User, *StaffProfile, *Role, error) {
	if err := auth.ValidatePassword(req.Password); err != nil {
		return nil, nil, nil, err
	}

	if req.Role != nil {
		if !authz.CanGrantRole(actorRole.Level, actorRole.ScopeType, req.Role.Level, req.Role.ScopeType) {
			return nil, nil, nil, ErrCannotManageHigherRole
		}
		if err := s.validateRoleScope(ctx, req.Role.ScopeType, req.Role.ScopeID); err != nil {
			return nil, nil, nil, err
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
		Email:         req.Email,
		PasswordHash:  passwordHash,
		FullNameEN:    req.FullNameEN,
		FullNameLocal: req.FullNameLocal,
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
			TitleEN:    req.Role.TitleEN,
			TitleLocal: req.Role.TitleLocal,
			Level:      req.Role.Level,
			ScopeType:  req.Role.ScopeType,
			ScopeID:    req.Role.ScopeID,
			Domain:     req.Role.Domain,
			AssignedBy: &adminID,
		}
	}

	if err := s.repo.CreateStaffUserTx(ctx, user, profile, role); err != nil {
		return nil, nil, nil, err
	}

	return user, profile, role, nil
}

func (s *Service) AdminSetPassword(ctx context.Context, userID uuid.UUID, password string) error {
	if err := auth.ValidatePassword(password); err != nil {
		return err
	}

	if _, err := s.repo.GetUser(ctx, userID); err != nil {
		return err
	}

	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}

	if err := s.repo.SetPassword(ctx, userID, passwordHash); err != nil {
		return err
	}

	if err := s.tokens.DeleteUserTokens(ctx, userID); err != nil {
		return err
	}

	if s.notifier != nil {
		body := "Your password has been reset by an administrator. Please log in with your new password."
		_ = s.notifier.Send(ctx, userID, "password_reset", "Password Reset", &body, nil)
	}

	return nil
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

	if err := auth.ValidatePassword(req.NewPassword); err != nil {
		return err
	}

	passwordHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		return err
	}

	if err := s.repo.SetPassword(ctx, userID, passwordHash); err != nil {
		return err
	}

	return s.tokens.DeleteUserTokens(ctx, userID)
}

func (s *Service) AssignRole(ctx context.Context, adminID, targetUserID uuid.UUID, actorRole *auth.RoleClaim, req AssignRoleRequest) (*Role, error) {
	if adminID == targetUserID {
		return nil, ErrCannotModifyOwnRole
	}

	targetClaim := &auth.RoleClaim{
		Level:     req.Level,
		ScopeType: req.ScopeType,
		ScopeID:   req.ScopeID,
	}
	if !s.roles.CanManageRole(ctx, actorRole, targetClaim) {
		return nil, ErrCannotManageHigherRole
	}

	if err := s.validateRoleScope(ctx, req.ScopeType, req.ScopeID); err != nil {
		return nil, err
	}

	if _, err := s.repo.GetUser(ctx, targetUserID); err != nil {
		return nil, err
	}

	role := &Role{
		UserID:     targetUserID,
		TitleEN:    req.TitleEN,
		TitleLocal: req.TitleLocal,
		Level:      req.Level,
		ScopeType:  req.ScopeType,
		ScopeID:    req.ScopeID,
		Domain:     req.Domain,
		AssignedBy: &adminID,
	}

	if err := s.repo.CreateRole(ctx, role); err != nil {
		if errors.Is(err, ErrRoleExists) {
			if err := s.repo.UpdateRole(ctx, role); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	_ = s.tokens.DeleteUserTokens(ctx, targetUserID)

	if s.notifier != nil {
		body := "You have been assigned the role: " + role.Level + " (" + role.ScopeType + ")"
		_ = s.notifier.Send(ctx, targetUserID, "role_assigned", "Role Assigned", &body, map[string]any{
			"role_id":    role.ID,
			"permission": role.Level,
			"scope_type": role.ScopeType,
		})
	}

	return role, nil
}

func (s *Service) RemoveRole(ctx context.Context, adminID, targetUserID uuid.UUID, actorRole *auth.RoleClaim) error {
	if adminID == targetUserID {
		return ErrCannotModifyOwnRole
	}

	targetRole, err := s.repo.GetRole(ctx, targetUserID)
	if err != nil {
		return err
	}
	if targetRole == nil {
		return ErrRoleNotFound
	}

	targetClaim := &auth.RoleClaim{
		Level:     targetRole.Level,
		ScopeType: targetRole.ScopeType,
		ScopeID:   targetRole.ScopeID,
	}
	if !s.roles.CanManageRole(ctx, actorRole, targetClaim) {
		return ErrCannotManageHigherRole
	}

	if err := s.repo.DeleteRole(ctx, targetUserID); err != nil {
		return err
	}

	_ = s.tokens.DeleteUserTokens(ctx, targetUserID)

	if s.notifier != nil {
		body := "Your role has been removed."
		_ = s.notifier.Send(ctx, targetUserID, "role_removed", "Role Removed", &body, nil)
	}

	return nil
}

func (s *Service) validateRoleScope(ctx context.Context, scopeType string, scopeID *uuid.UUID) error {
	if scopeType == "platform" || scopeType == "university" {
		if scopeID != nil {
			return ErrScopeIDNotAllowed
		}
		return nil
	}

	if scopeID == nil {
		return ErrScopeIDRequired
	}

	exists, err := s.repo.ScopeExists(ctx, scopeType, *scopeID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrInvalidScopeID
	}
	return nil
}
