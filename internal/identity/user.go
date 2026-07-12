package identity

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/shared/pagination"
)

// ── Entities ───────────────────────────────────────────────────────────────

// User is a person's account: credentials, display identity, and account
// state. The users table is the one cross-context published table (§19a):
// other contexts may read its display columns.
type User struct {
	ID                uuid.UUID `db:"id"`
	Email             string    `db:"email"`
	PasswordHash      string    `db:"password_hash"`
	FullNameEN        string    `db:"full_name_en"`
	FullNameLocal     *string   `db:"full_name_local"`
	AvatarURL         *string   `db:"avatar_url"`
	Phone             *string   `db:"phone"`
	IsActive          bool      `db:"is_active"`
	IsVerified        bool      `db:"is_verified"`
	PreferredLanguage Language  `db:"preferred_language"`
	Timezone          string    `db:"timezone"`
	Theme             Theme     `db:"theme"`
	CreatedAt         time.Time `db:"created_at"`
	UpdatedAt         time.Time `db:"updated_at"`
}

// Session is a live login as shown to its owner: device, address, and the
// refresh-token lifetime that bounds it.
type Session struct {
	ID        uuid.UUID  `db:"id"`
	Device    *string    `db:"device"`
	IPAddress *string    `db:"ip_address"`
	CreatedAt time.Time  `db:"created_at"`
	ExpiresAt time.Time  `db:"expires_at"`
	UsedAt    *time.Time `db:"used_at"`
}

// UserFilters narrows admin user listings; nil fields do not filter.
type UserFilters struct {
	IsActive        *bool
	HasStaffProfile *bool
	HasRole         *bool
}

// ── Inputs ─────────────────────────────────────────────────────────────────

// UpdateProfileInput is a partial self-service profile edit; nil fields are
// left unchanged, and an empty string clears an optional field.
type UpdateProfileInput struct {
	FullNameEN    *string
	FullNameLocal *string
	AvatarURL     *string
	Phone         *string
}

func derefInt(p *int, def int) int {
	if p == nil {
		return def
	}
	return *p
}

// ── Ports ──────────────────────────────────────────────────────────────────

// Notifier sends a notification (communication context satisfies it).
type Notifier interface {
	Send(ctx context.Context, userID uuid.UUID, notifType, title string, body *string, data map[string]any) error
}

// RoleManager authorizes role grants/management (authz context satisfies it).
// Defined as a port so identity does NOT import authz (avoids an import cycle:
// authz already imports identity for RoleClaim).
type RoleManager interface {
	CanManageRole(ctx context.Context, actor, target *RoleClaim) bool
	CanGrantRole(actor, target *RoleClaim) bool
}

// StudentReader / UniversityReader use identity-local result types (not
// student./university. types) to avoid an import cycle — those packages import
// authz, which imports identity. Composition root adapts the real services.

// StudentInfo is the slim student projection identity needs for scope
// resolution. Student records key on the account id, so the record's
// existence is the only identity beyond the user's own.
type StudentInfo struct {
	ProgramID uuid.UUID
}

// ProgramInfo is the slim programme projection identity needs.
type ProgramInfo struct {
	ID           uuid.UUID
	NameEN       string
	NameLocal    *string
	DepartmentID uuid.UUID
}

// DepartmentInfo is the slim department projection identity needs.
type DepartmentInfo struct {
	ID        uuid.UUID
	NameEN    string
	NameLocal *string
	CollegeID uuid.UUID
}

// CollegeInfo is the slim college projection identity needs.
type CollegeInfo struct {
	ID        uuid.UUID
	NameEN    string
	NameLocal *string
}

// StudentReader resolves the student record behind a user account.
// GetStudentByUserID returns (nil, nil) when the user is not a student.
type StudentReader interface {
	GetStudentByUserID(ctx context.Context, userID uuid.UUID) (*StudentInfo, error)
}

// UniversityReader resolves institutional-hierarchy names for scope display.
// Get methods return (nil, nil) when the entity does not exist.
type UniversityReader interface {
	GetProgram(ctx context.Context, id uuid.UUID) (*ProgramInfo, error)
	GetDepartment(ctx context.Context, id uuid.UUID) (*DepartmentInfo, error)
	GetCollege(ctx context.Context, id uuid.UUID) (*CollegeInfo, error)
	ListActiveColleges(ctx context.Context) ([]CollegeInfo, error)
}

// CourseRoleReader returns the user's teacher record id (nil if not a teacher)
// plus their ACTIVE course memberships (teaching + enrolled). Bounded —
// historical memberships belong to a separate paginated endpoint.
type CourseRoleReader interface {
	CourseRolesForUser(ctx context.Context, userID uuid.UUID) (*CourseMemberships, error)
}

// UserRepository is the user aggregate's store: account fields, credentials,
// the institutional role, and the staff profile. Get methods return the noun's
// not-found sentinel; Create methods return the noun's already-exists sentinel
// on a unique-constraint violation (users.email, roles.user_id,
// staff_profiles.user_id). CreateStaffUserTx is atomic: user + staff profile
// (+ optional role) are created in one transaction or not at all.
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
	GetRolesForUsers(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]*Role, error)
	// SetRole grants or replaces the user's single role in one atomic
	// statement (an upsert on roles.user_id), so concurrent assignments
	// cannot race a create against an update.
	SetRole(ctx context.Context, role *Role) error
	DeleteRole(ctx context.Context, userID uuid.UUID) error
	GetStaffProfile(ctx context.Context, userID uuid.UUID) (*StaffProfile, error)
	CreateStaffProfile(ctx context.Context, profile *StaffProfile) error
	// UpdateStaffProfile merges the non-nil input fields onto the stored row
	// in one atomic statement, so concurrent partial edits cannot lose each
	// other's fields through a stale read.
	UpdateStaffProfile(ctx context.Context, userID uuid.UUID, in StaffProfileInput) (*StaffProfile, error)
	ScopeExists(ctx context.Context, scopeType string, scopeID uuid.UUID) (bool, error)
	CreateStaffUserTx(ctx context.Context, user *User, profile *StaffProfile, role *Role) error
}

// ── UserService (use cases) ────────────────────────────────────────────────

// UserService is account self-service plus the admin surface for users,
// roles, and staff profiles.
type UserService struct {
	repo       UserRepository
	tokens     TokenRepository
	notifier   Notifier
	roles      RoleManager
	students   StudentReader
	university UniversityReader
	courses    CourseRoleReader
	log        *slog.Logger
}

// NewUserService wires the user use cases.
func NewUserService(repo UserRepository, tokens TokenRepository, notifier Notifier, roles RoleManager, students StudentReader, university UniversityReader, courses CourseRoleReader, log *slog.Logger) *UserService {
	return &UserService{repo: repo, tokens: tokens, notifier: notifier, roles: roles, students: students, university: university, courses: courses, log: log}
}

// GetProfile returns the caller's own account.
func (s *UserService) GetProfile(ctx context.Context, userID uuid.UUID) (*User, error) {
	return s.repo.GetUser(ctx, userID)
}

// UpdateProfile applies a partial edit to the caller's own account and returns
// the result.
func (s *UserService) UpdateProfile(ctx context.Context, userID uuid.UUID, in UpdateProfileInput) (*User, error) {
	user, err := s.repo.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if in.FullNameEN != nil {
		user.FullNameEN = *in.FullNameEN
	}
	if in.FullNameLocal != nil {
		user.FullNameLocal = emptyToNil(in.FullNameLocal)
	}
	if in.AvatarURL != nil {
		user.AvatarURL = emptyToNil(in.AvatarURL)
	}
	if in.Phone != nil {
		user.Phone = emptyToNil(in.Phone)
	}
	if err := s.repo.Update(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func emptyToNil(p *string) *string {
	if p == nil || *p == "" {
		return nil
	}
	return p
}

// UpdateEmail changes the caller's email after re-verifying their password.
// It returns ErrSameEmail, ErrInvalidPassword, or ErrEmailExists on the
// respective violations.
func (s *UserService) UpdateEmail(ctx context.Context, userID uuid.UUID, email, password string) error {
	user, err := s.repo.GetUser(ctx, userID)
	if err != nil {
		return err
	}
	if user.Email == email {
		return ErrSameEmail
	}
	hash, err := s.repo.GetPasswordHash(ctx, userID)
	if err != nil {
		return err
	}
	if !CheckPassword(password, hash) {
		return ErrInvalidPassword
	}
	// Friendly pre-check only; the users.email UNIQUE constraint decides races.
	exists, err := s.repo.EmailExists(ctx, email)
	if err != nil {
		return err
	}
	if exists {
		return ErrEmailExists
	}
	return s.repo.UpdateEmail(ctx, userID, email)
}

// ListUsers pages through users matching the filters, newest first.
func (s *UserService) ListUsers(ctx context.Context, params pagination.PageParams, filters UserFilters) ([]User, bool, error) {
	return s.repo.List(ctx, params, filters)
}

// GetUserByID returns one user for the admin surface.
func (s *UserService) GetUserByID(ctx context.Context, userID uuid.UUID) (*User, error) {
	return s.repo.GetUser(ctx, userID)
}

// DeactivateUser disables the account and revokes every session, so the lockout
// takes effect at the next token refresh rather than at token expiry.
func (s *UserService) DeactivateUser(ctx context.Context, userID uuid.UUID) error {
	if err := s.repo.Deactivate(ctx, userID); err != nil {
		return err
	}
	return s.tokens.DeleteUserTokens(ctx, userID)
}

// GetSessions lists the caller's live sessions.
func (s *UserService) GetSessions(ctx context.Context, userID uuid.UUID) ([]Session, error) {
	tokens, err := s.tokens.GetUserSessions(ctx, userID)
	if err != nil {
		return nil, err
	}
	sessions := make([]Session, len(tokens))
	for i, t := range tokens {
		sessions[i] = Session{ID: t.ID, Device: t.Device, IPAddress: t.IPAddress, CreatedAt: t.CreatedAt, ExpiresAt: t.ExpiresAt, UsedAt: t.UsedAt}
	}
	return sessions, nil
}

// RevokeSession ends one of the caller's sessions. It returns
// ErrSessionNotFound when the session is not among the caller's own — scoping
// the lookup to the caller's tokens is what stops cross-user revocation.
func (s *UserService) RevokeSession(ctx context.Context, userID, sessionID uuid.UUID) error {
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

// RevokeOtherSessions ends every session of the caller except the one to keep.
func (s *UserService) RevokeOtherSessions(ctx context.Context, userID, keepSessionID uuid.UUID) error {
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

// AdminSetPassword resets a user's password on their behalf and revokes every
// session, forcing a fresh login with the new password.
func (s *UserService) AdminSetPassword(ctx context.Context, userID uuid.UUID, password string) error {
	if err := ValidatePassword(password); err != nil {
		return err
	}
	if _, err := s.repo.GetUser(ctx, userID); err != nil {
		return err
	}
	passwordHash, err := HashPassword(password)
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
		if err := s.notifier.Send(ctx, userID, "password_reset", "Password Reset", &body, nil); err != nil {
			s.log.WarnContext(ctx, "password reset notification failed", "user_id", userID, "error", err)
		}
	}
	return nil
}

// ChangePassword changes the caller's own password after verifying the current
// one, and revokes every session so all devices re-authenticate with the new
// password. It returns ErrInvalidPassword,
// ErrSamePassword, or the password-policy sentinels on violation.
func (s *UserService) ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error {
	hash, err := s.repo.GetPasswordHash(ctx, userID)
	if err != nil {
		return err
	}
	if !CheckPassword(currentPassword, hash) {
		return ErrInvalidPassword
	}
	if currentPassword == newPassword {
		return ErrSamePassword
	}
	if err := ValidatePassword(newPassword); err != nil {
		return err
	}
	passwordHash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}
	if err := s.repo.SetPassword(ctx, userID, passwordHash); err != nil {
		return err
	}
	return s.tokens.DeleteUserTokens(ctx, userID)
}
