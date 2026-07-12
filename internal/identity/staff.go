package identity

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ── Entity ─────────────────────────────────────────────────────────────────

// StaffProfile is the employment record attached to a staff member's
// account, keyed by the account (staff_profiles PK = user_id).
type StaffProfile struct {
	UserID         uuid.UUID `db:"user_id"`
	HighestDegree  *string   `db:"highest_degree"`
	FieldOfStudy   *string   `db:"field_of_study"`
	YearsOfService int       `db:"years_of_service"`
	Salary         *string   `db:"salary"`
	SalaryCurrency *string   `db:"salary_currency"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}

// StaffProfileInput is a partial staff-profile write; nil fields are left
// unchanged (or defaulted on create).
type StaffProfileInput struct {
	HighestDegree  *string
	FieldOfStudy   *string
	YearsOfService *int
	Salary         *string // already formatted by the transport
	SalaryCurrency *string
}

// CreateStaffUserInput is what admin staff-account creation needs: the
// account, its staff profile, and optionally an institutional role.
type CreateStaffUserInput struct {
	Email         string
	Password      string
	FullNameEN    string
	FullNameLocal *string
	StaffProfile  StaffProfileInput
	Role          *RoleInput
}

// ── Use cases ──────────────────────────────────────────────────────────────

// GetStaffProfile returns the user's staff profile, or
// ErrStaffProfileNotFound if they have none.
func (s *UserService) GetStaffProfile(ctx context.Context, userID uuid.UUID) (*StaffProfile, error) {
	return s.repo.GetStaffProfile(ctx, userID)
}

// CreateStaffProfile attaches a staff profile to an existing user. It returns
// ErrStaffProfileExists when the user already has one.
func (s *UserService) CreateStaffProfile(ctx context.Context, userID uuid.UUID, in StaffProfileInput) (*StaffProfile, error) {
	if _, err := s.repo.GetUser(ctx, userID); err != nil {
		return nil, err
	}
	// Friendly pre-check only; the staff_profiles.user_id UNIQUE constraint
	// decides races.
	if _, err := s.repo.GetStaffProfile(ctx, userID); err == nil {
		return nil, ErrStaffProfileExists
	} else if !errors.Is(err, ErrStaffProfileNotFound) {
		return nil, err
	}
	profile := &StaffProfile{
		UserID:         userID,
		HighestDegree:  in.HighestDegree,
		FieldOfStudy:   in.FieldOfStudy,
		YearsOfService: derefInt(in.YearsOfService, 0),
		Salary:         in.Salary,
		SalaryCurrency: in.SalaryCurrency,
	}
	if err := s.repo.CreateStaffProfile(ctx, profile); err != nil {
		return nil, err
	}
	return profile, nil
}

// UpdateStaffProfile applies a partial edit to the user's staff profile and
// returns the result, or ErrStaffProfileNotFound if they have none. The merge
// happens in the repository's single UPDATE, so concurrent edits to different
// fields never clobber each other through a stale read.
func (s *UserService) UpdateStaffProfile(ctx context.Context, userID uuid.UUID, in StaffProfileInput) (*StaffProfile, error) {
	return s.repo.UpdateStaffProfile(ctx, userID, in)
}

// CreateStaffUser creates a staff account — user, staff profile, and
// optionally an institutional role — atomically: all three exist afterwards
// or none do. It returns ErrEmailExists when the email is taken,
// ErrCannotManageHigherRole when the role exceeds the actor's authority, and
// the scope-validation or password-policy sentinels on those violations.
func (s *UserService) CreateStaffUser(ctx context.Context, adminID uuid.UUID, actorRole *RoleClaim, in CreateStaffUserInput) (*User, *StaffProfile, *Role, error) {
	if err := ValidatePassword(in.Password); err != nil {
		return nil, nil, nil, err
	}
	if in.Role != nil {
		targetClaim := &RoleClaim{Level: in.Role.Level, ScopeType: in.Role.ScopeType, ScopeID: in.Role.ScopeID}
		if !s.roles.CanGrantRole(actorRole, targetClaim) {
			return nil, nil, nil, ErrCannotManageHigherRole
		}
		if err := s.validateRoleScope(ctx, in.Role.ScopeType, in.Role.ScopeID); err != nil {
			return nil, nil, nil, err
		}
	}
	// Friendly pre-check only; the users.email UNIQUE constraint decides races.
	exists, err := s.repo.EmailExists(ctx, in.Email)
	if err != nil {
		return nil, nil, nil, err
	}
	if exists {
		return nil, nil, nil, ErrEmailExists
	}
	passwordHash, err := HashPassword(in.Password)
	if err != nil {
		return nil, nil, nil, err
	}

	user := &User{Email: in.Email, PasswordHash: passwordHash, FullNameEN: in.FullNameEN, FullNameLocal: in.FullNameLocal}
	profile := &StaffProfile{
		HighestDegree:  in.StaffProfile.HighestDegree,
		FieldOfStudy:   in.StaffProfile.FieldOfStudy,
		YearsOfService: derefInt(in.StaffProfile.YearsOfService, 0),
		Salary:         in.StaffProfile.Salary,
		SalaryCurrency: in.StaffProfile.SalaryCurrency,
	}
	var role *Role
	if in.Role != nil {
		role = &Role{
			TitleEN:    in.Role.TitleEN,
			TitleLocal: in.Role.TitleLocal,
			Level:      in.Role.Level,
			ScopeType:  in.Role.ScopeType,
			ScopeID:    in.Role.ScopeID,
			Domain:     in.Role.Domain,
			AssignedBy: &adminID,
		}
	}

	if err := s.repo.CreateStaffUserTx(ctx, user, profile, role); err != nil {
		return nil, nil, nil, err
	}
	return user, profile, role, nil
}
