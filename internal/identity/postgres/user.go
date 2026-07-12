package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/randdotdev/e-campus-server/internal/identity"
	"github.com/randdotdev/e-campus-server/internal/shared/pagination"
)

// Repo backs both identity.UserRepository and identity.AuthUserStore.
type Repo struct {
	db *sqlx.DB
}

var (
	_ identity.UserRepository = (*Repo)(nil)
	_ identity.AuthUserStore  = (*Repo)(nil)
)

// NewRepository wires the SQL adapter for users, roles, and staff profiles.
func NewRepository(db *sqlx.DB) *Repo { return &Repo{db: db} }

// ── AuthUserStore ──────────────────────────────────────────────────────────

// Create inserts a user account. The users.email UNIQUE constraint is the
// duplicate guard; its violation surfaces as ErrEmailExists.
func (r *Repo) Create(ctx context.Context, email, passwordHash, fullNameEN string, fullNameLocal *string) (*identity.UserData, error) {
	var u identity.User
	query := `
		INSERT INTO users (email, password_hash, full_name_en, full_name_local)
		VALUES ($1, $2, $3, $4)
		RETURNING id, email, password_hash, full_name_en, full_name_local, avatar_url, is_active, is_verified, preferred_language, timezone, theme, created_at`
	err := r.db.QueryRowxContext(ctx, query, email, passwordHash, fullNameEN, fullNameLocal).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.FullNameEN, &u.FullNameLocal,
		&u.AvatarURL, &u.IsActive, &u.IsVerified, &u.PreferredLanguage, &u.Timezone, &u.Theme, &u.CreatedAt,
	)
	if isUniqueViolation(err) {
		return nil, identity.ErrEmailExists
	}
	if err != nil {
		return nil, err
	}
	return toUserData(&u), nil
}

// GetByEmail fetches the credentials view of a user by email.
func (r *Repo) GetByEmail(ctx context.Context, email string) (*identity.UserData, error) {
	var u identity.User
	if err := r.db.GetContext(ctx, &u, `SELECT * FROM users WHERE email = $1`, email); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, identity.ErrUserNotFound
		}
		return nil, err
	}
	return toUserData(&u), nil
}

// GetByID fetches the credentials view of a user by id.
func (r *Repo) GetByID(ctx context.Context, id uuid.UUID) (*identity.UserData, error) {
	var u identity.User
	if err := r.db.GetContext(ctx, &u, `SELECT * FROM users WHERE id = $1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, identity.ErrUserNotFound
		}
		return nil, err
	}
	return toUserData(&u), nil
}

// EmailExists reports whether any user holds the email. Advisory only — the
// UNIQUE constraint on users.email decides races.
func (r *Repo) EmailExists(ctx context.Context, email string) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists, `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`, email)
	return exists, err
}

// GetUserRole returns auth's projection of the user's unexpired role, or nil
// if they have none.
func (r *Repo) GetUserRole(ctx context.Context, userID uuid.UUID) (*identity.RoleData, error) {
	var role identity.Role
	query := `SELECT * FROM roles WHERE user_id = $1 AND (expires_at IS NULL OR expires_at > NOW())`
	if err := r.db.GetContext(ctx, &role, query, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return toRoleData(&role), nil
}

func toUserData(u *identity.User) *identity.UserData {
	return &identity.UserData{
		ID: u.ID, Email: u.Email, PasswordHash: u.PasswordHash, FullNameEN: u.FullNameEN,
		FullNameLocal: u.FullNameLocal, AvatarURL: u.AvatarURL, IsActive: u.IsActive,
		IsVerified: u.IsVerified, CreatedAt: u.CreatedAt,
	}
}

func toRoleData(r *identity.Role) *identity.RoleData {
	return &identity.RoleData{
		ID: r.ID, TitleEN: r.TitleEN, TitleLocal: r.TitleLocal, Level: r.Level,
		ScopeType: r.ScopeType, ScopeID: r.ScopeID, Domain: r.Domain,
	}
}

// ── UserRepository ─────────────────────────────────────────────────────────

// GetUser fetches one user.
func (r *Repo) GetUser(ctx context.Context, id uuid.UUID) (*identity.User, error) {
	var u identity.User
	if err := r.db.GetContext(ctx, &u, `SELECT * FROM users WHERE id = $1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, identity.ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}

// Update writes the user's profile fields.
func (r *Repo) Update(ctx context.Context, user *identity.User) error {
	query := `
		UPDATE users SET full_name_en = $2, full_name_local = $3, avatar_url = $4, phone = $5
		WHERE id = $1 RETURNING updated_at`
	err := r.db.QueryRowxContext(ctx, query, user.ID, user.FullNameEN, user.FullNameLocal, user.AvatarURL, user.Phone).Scan(&user.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return identity.ErrUserNotFound
	}
	return err
}

// UpdateEmail changes the user's email. The users.email UNIQUE constraint is
// the duplicate guard; its violation surfaces as ErrEmailExists.
func (r *Repo) UpdateEmail(ctx context.Context, id uuid.UUID, email string) error {
	err := r.affectOne(ctx, `UPDATE users SET email = $2 WHERE id = $1`, id, email)
	if isUniqueViolation(err) {
		return identity.ErrEmailExists
	}
	return err
}

// List pages through users matching the filters, newest first.
func (r *Repo) List(ctx context.Context, params pagination.PageParams, filters identity.UserFilters) ([]identity.User, bool, error) {
	var conditions []string
	var args []any
	argN := 1

	if params.Cursor != "" {
		createdAt, id, err := pagination.DecodeCursor(params.Cursor)
		if err != nil {
			return nil, false, err
		}
		conditions = append(conditions, fmt.Sprintf("(created_at, id) < ($%d, $%d)", argN, argN+1))
		args = append(args, createdAt, id)
		argN += 2
	}
	if params.Query != "" {
		conditions = append(conditions, fmt.Sprintf("(email ILIKE $%d OR full_name_en ILIKE $%d OR full_name_local ILIKE $%d)", argN, argN, argN))
		args = append(args, "%"+pagination.EscapeLike(params.Query)+"%")
		argN++
	}
	if filters.IsActive != nil {
		conditions = append(conditions, fmt.Sprintf("is_active = $%d", argN))
		args = append(args, *filters.IsActive)
		argN++
	}
	if filters.HasStaffProfile != nil {
		if *filters.HasStaffProfile {
			conditions = append(conditions, "EXISTS (SELECT 1 FROM staff_profiles WHERE staff_profiles.user_id = users.id)")
		} else {
			conditions = append(conditions, "NOT EXISTS (SELECT 1 FROM staff_profiles WHERE staff_profiles.user_id = users.id)")
		}
	}
	if filters.HasRole != nil && *filters.HasRole {
		conditions = append(conditions, "EXISTS (SELECT 1 FROM roles WHERE roles.user_id = users.id)")
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}
	query := fmt.Sprintf("SELECT * FROM users %s ORDER BY created_at DESC, id DESC LIMIT $%d", where, argN)
	args = append(args, params.Limit+1)

	var users []identity.User
	if err := r.db.SelectContext(ctx, &users, query, args...); err != nil {
		return nil, false, err
	}
	hasMore := len(users) > params.Limit
	if hasMore {
		users = users[:params.Limit]
	}
	return users, hasMore, nil
}

// Deactivate disables the user's account.
func (r *Repo) Deactivate(ctx context.Context, id uuid.UUID) error {
	return r.affectOne(ctx, `UPDATE users SET is_active = false WHERE id = $1`, id)
}

// GetPasswordHash fetches the user's password hash for verification.
func (r *Repo) GetPasswordHash(ctx context.Context, id uuid.UUID) (string, error) {
	var hash string
	if err := r.db.GetContext(ctx, &hash, `SELECT password_hash FROM users WHERE id = $1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", identity.ErrUserNotFound
		}
		return "", err
	}
	return hash, nil
}

// SetPassword writes a new password hash.
func (r *Repo) SetPassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	return r.affectOne(ctx, `UPDATE users SET password_hash = $2 WHERE id = $1`, userID, passwordHash)
}

// GetRolesForUsers returns the unexpired roles of the given users keyed by
// user id; users without a role are absent from the map.
func (r *Repo) GetRolesForUsers(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]*identity.Role, error) {
	if len(userIDs) == 0 {
		return map[uuid.UUID]*identity.Role{}, nil
	}
	placeholders := make([]string, len(userIDs))
	args := make([]any, len(userIDs))
	for i, id := range userIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	query := fmt.Sprintf("SELECT * FROM roles WHERE user_id IN (%s) AND (expires_at IS NULL OR expires_at > NOW())", strings.Join(placeholders, ", "))
	var roles []identity.Role
	if err := r.db.SelectContext(ctx, &roles, query, args...); err != nil {
		return nil, err
	}
	result := make(map[uuid.UUID]*identity.Role, len(roles))
	for i := range roles {
		result[roles[i].UserID] = &roles[i]
	}
	return result, nil
}

// GetRole returns the user's unexpired role, or nil if they have none.
func (r *Repo) GetRole(ctx context.Context, userID uuid.UUID) (*identity.Role, error) {
	var role identity.Role
	query := `SELECT * FROM roles WHERE user_id = $1 AND (expires_at IS NULL OR expires_at > NOW())`
	if err := r.db.GetContext(ctx, &role, query, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &role, nil
}

// GetStaffProfile fetches the user's staff profile.
func (r *Repo) GetStaffProfile(ctx context.Context, userID uuid.UUID) (*identity.StaffProfile, error) {
	var p identity.StaffProfile
	if err := r.db.GetContext(ctx, &p, `SELECT * FROM staff_profiles WHERE user_id = $1`, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, identity.ErrStaffProfileNotFound
		}
		return nil, err
	}
	return &p, nil
}

// CreateStaffProfile inserts a staff profile. The staff_profiles.user_id
// UNIQUE constraint is the duplicate guard; its violation surfaces as
// ErrStaffProfileExists.
func (r *Repo) CreateStaffProfile(ctx context.Context, p *identity.StaffProfile) error {
	query := `
		INSERT INTO staff_profiles (user_id, highest_degree, field_of_study, years_of_service, salary, salary_currency)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING created_at, updated_at`
	err := r.db.QueryRowxContext(ctx, query, p.UserID, p.HighestDegree, p.FieldOfStudy, p.YearsOfService, p.Salary, p.SalaryCurrency).
		Scan(&p.CreatedAt, &p.UpdatedAt)
	if isUniqueViolation(err) {
		return identity.ErrStaffProfileExists
	}
	return err
}

// UpdateStaffProfile merges the non-nil input fields onto the stored row in
// one UPDATE — nil fields keep their stored value, so concurrent partial
// edits to different fields compose instead of clobbering each other.
func (r *Repo) UpdateStaffProfile(ctx context.Context, userID uuid.UUID, in identity.StaffProfileInput) (*identity.StaffProfile, error) {
	query := `
		UPDATE staff_profiles
		SET highest_degree = COALESCE($2, highest_degree),
		    field_of_study = COALESCE($3, field_of_study),
		    years_of_service = COALESCE($4, years_of_service),
		    salary = COALESCE($5, salary),
		    salary_currency = COALESCE($6, salary_currency)
		WHERE user_id = $1
		RETURNING *`
	var p identity.StaffProfile
	err := r.db.QueryRowxContext(ctx, query, userID, in.HighestDegree, in.FieldOfStudy, in.YearsOfService, in.Salary, in.SalaryCurrency).StructScan(&p)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, identity.ErrStaffProfileNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// ScopeExists reports whether the scope target (college, department, or
// programme) exists. Read-only cross-context lookup.
func (r *Repo) ScopeExists(ctx context.Context, scopeType string, scopeID uuid.UUID) (bool, error) {
	var query string
	switch scopeType {
	case "college":
		query = `SELECT EXISTS(SELECT 1 FROM colleges WHERE id = $1)`
	case "department":
		query = `SELECT EXISTS(SELECT 1 FROM departments WHERE id = $1)`
	case "program":
		query = `SELECT EXISTS(SELECT 1 FROM programs WHERE id = $1)`
	default:
		return false, identity.ErrInvalidScopeID
	}
	var exists bool
	err := r.db.GetContext(ctx, &exists, query, scopeID)
	return exists, err
}

// CreateStaffUserTx creates a staff account — user, staff profile, and
// optionally a role — in one transaction. A duplicate email surfaces as
// ErrEmailExists (the profile and role rows key on the new user, so the email
// is the only unique constraint that can fire).
func (r *Repo) CreateStaffUserTx(ctx context.Context, user *identity.User, profile *identity.StaffProfile, role *identity.Role) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	userQuery := `
		INSERT INTO users (email, password_hash, full_name_en, full_name_local)
		VALUES ($1, $2, $3, $4) RETURNING *`
	if err := tx.QueryRowxContext(ctx, userQuery, user.Email, user.PasswordHash, user.FullNameEN, user.FullNameLocal).StructScan(user); err != nil {
		if isUniqueViolation(err) {
			return identity.ErrEmailExists
		}
		return err
	}

	profile.UserID = user.ID
	profileQuery := `
		INSERT INTO staff_profiles (user_id, highest_degree, field_of_study, years_of_service, salary, salary_currency)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING created_at, updated_at`
	if err := tx.QueryRowxContext(ctx, profileQuery, profile.UserID, profile.HighestDegree, profile.FieldOfStudy, profile.YearsOfService, profile.Salary, profile.SalaryCurrency).
		Scan(&profile.CreatedAt, &profile.UpdatedAt); err != nil {
		return err
	}

	if role != nil {
		role.UserID = user.ID
		roleQuery := `
			INSERT INTO roles (user_id, title_en, title_local, level, scope_type, scope_id, assigned_by, domain)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id, created_at`
		if err := tx.QueryRowxContext(ctx, roleQuery, role.UserID, role.TitleEN, role.TitleLocal, role.Level, role.ScopeType, role.ScopeID, role.AssignedBy, role.Domain).
			Scan(&role.ID, &role.CreatedAt); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// SetRole grants or replaces the user's single role in one atomic statement:
// an upsert on the roles.user_id UNIQUE constraint, so concurrent assignments
// cannot race a create against an update.
func (r *Repo) SetRole(ctx context.Context, role *identity.Role) error {
	query := `
		INSERT INTO roles (user_id, title_en, title_local, level, scope_type, scope_id, assigned_by, expires_at, domain)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (user_id) DO UPDATE
		SET title_en = EXCLUDED.title_en,
			title_local = EXCLUDED.title_local,
			level = EXCLUDED.level,
			scope_type = EXCLUDED.scope_type,
			scope_id = EXCLUDED.scope_id,
			assigned_by = EXCLUDED.assigned_by,
			expires_at = EXCLUDED.expires_at,
			domain = EXCLUDED.domain
		RETURNING id, created_at`
	return r.db.QueryRowxContext(ctx, query, role.UserID, role.TitleEN, role.TitleLocal, role.Level, role.ScopeType, role.ScopeID, role.AssignedBy, role.ExpiresAt, role.Domain).
		Scan(&role.ID, &role.CreatedAt)
}

// DeleteRole removes the user's role, returning ErrRoleNotFound if they had none.
func (r *Repo) DeleteRole(ctx context.Context, userID uuid.UUID) error {
	query := `DELETE FROM roles WHERE user_id = $1`
	result, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return identity.ErrRoleNotFound
	}
	return nil
}

func (r *Repo) affectOne(ctx context.Context, query string, args ...any) error {
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return identity.ErrUserNotFound
	}
	return nil
}
