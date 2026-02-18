package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/ranjdotdev/e-campus-server/internal/auth"
	"github.com/ranjdotdev/e-campus-server/internal/pagination"
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, email, passwordHash, fullNameEN string, fullNameKU *string) (*auth.UserData, error) {
	var user User
	query := `
		INSERT INTO users (email, password_hash, full_name_en, full_name_ku)
		VALUES ($1, $2, $3, $4)
		RETURNING id, email, password_hash, full_name_en, full_name_ku, avatar_url, is_active, is_verified, created_at`

	err := r.db.QueryRowxContext(ctx, query, email, passwordHash, fullNameEN, fullNameKU).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.FullNameEN, &user.FullNameKU,
		&user.AvatarURL, &user.IsActive, &user.IsVerified, &user.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return toUserData(&user), nil
}

func (r *Repository) GetByEmail(ctx context.Context, email string) (*auth.UserData, error) {
	var user User
	query := `SELECT * FROM users WHERE email = $1`

	if err := r.db.GetContext(ctx, &user, query, email); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, auth.ErrUserNotFound
		}
		return nil, err
	}
	return toUserData(&user), nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*auth.UserData, error) {
	var user User
	query := `SELECT * FROM users WHERE id = $1`

	if err := r.db.GetContext(ctx, &user, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, auth.ErrUserNotFound
		}
		return nil, err
	}
	return toUserData(&user), nil
}

func (r *Repository) EmailExists(ctx context.Context, email string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`
	err := r.db.GetContext(ctx, &exists, query, email)
	return exists, err
}

func (r *Repository) GetUserRole(ctx context.Context, userID uuid.UUID) (*auth.RoleData, error) {
	var role Role
	query := `
		SELECT * FROM roles
		WHERE user_id = $1
		AND (expires_at IS NULL OR expires_at > NOW())`

	if err := r.db.GetContext(ctx, &role, query, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return toRoleData(&role), nil
}

func toUserData(u *User) *auth.UserData {
	return &auth.UserData{
		ID:           u.ID,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		FullNameEN:   u.FullNameEN,
		FullNameKU:   u.FullNameKU,
		AvatarURL:    u.AvatarURL,
		IsActive:     u.IsActive,
		IsVerified:   u.IsVerified,
		CreatedAt:    u.CreatedAt,
	}
}

func toRoleData(r *Role) *auth.RoleData {
	return &auth.RoleData{
		ID:         r.ID,
		Title:      r.Title,
		Permission: r.Permission,
		ScopeType:  r.ScopeType,
		ScopeID:    r.ScopeID,
	}
}

func (r *Repository) GetUser(ctx context.Context, id uuid.UUID) (*User, error) {
	var user User
	query := `SELECT * FROM users WHERE id = $1`

	if err := r.db.GetContext(ctx, &user, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *Repository) Update(ctx context.Context, user *User) error {
	query := `
		UPDATE users
		SET full_name_en = $2, full_name_ku = $3, avatar_url = $4, phone = $5
		WHERE id = $1
		RETURNING updated_at`

	return r.db.QueryRowxContext(ctx, query,
		user.ID,
		user.FullNameEN,
		user.FullNameKU,
		user.AvatarURL,
		user.Phone,
	).Scan(&user.UpdatedAt)
}

func (r *Repository) UpdateEmail(ctx context.Context, id uuid.UUID, email string) error {
	query := `UPDATE users SET email = $2 WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id, email)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *Repository) List(ctx context.Context, params pagination.PageParams, filters UserFilters) ([]User, bool, error) {
	query := strings.Builder{}
	args := []any{}
	argN := 1

	query.WriteString("SELECT * FROM users WHERE 1=1")

	if params.Cursor != "" {
		createdAt, id, err := pagination.DecodeCursor(params.Cursor)
		if err != nil {
			return nil, false, err
		}
		query.WriteString(fmt.Sprintf(" AND (created_at, id) < ($%d, $%d)", argN, argN+1))
		args = append(args, createdAt, id)
		argN += 2
	}

	if params.Query != "" {
		query.WriteString(fmt.Sprintf(" AND (email ILIKE $%d OR full_name_en ILIKE $%d OR full_name_ku ILIKE $%d)", argN, argN, argN))
		args = append(args, "%"+pagination.EscapeLike(params.Query)+"%")
		argN++
	}

	if filters.IsActive != nil {
		query.WriteString(fmt.Sprintf(" AND is_active = $%d", argN))
		args = append(args, *filters.IsActive)
		argN++
	}

	if filters.HasStaffProfile != nil {
		if *filters.HasStaffProfile {
			query.WriteString(" AND EXISTS (SELECT 1 FROM staff_profiles WHERE staff_profiles.user_id = users.id)")
		} else {
			query.WriteString(" AND NOT EXISTS (SELECT 1 FROM staff_profiles WHERE staff_profiles.user_id = users.id)")
		}
	}

	query.WriteString(" ORDER BY created_at DESC, id DESC")
	query.WriteString(fmt.Sprintf(" LIMIT $%d", argN))
	args = append(args, params.Limit+1)

	var users []User
	if err := r.db.SelectContext(ctx, &users, query.String(), args...); err != nil {
		return nil, false, err
	}

	hasMore := len(users) > params.Limit
	if hasMore {
		users = users[:params.Limit]
	}

	return users, hasMore, nil
}

func (r *Repository) Deactivate(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE users SET is_active = false WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *Repository) GetPasswordHash(ctx context.Context, id uuid.UUID) (string, error) {
	var hash string
	query := `SELECT password_hash FROM users WHERE id = $1`

	if err := r.db.GetContext(ctx, &hash, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrUserNotFound
		}
		return "", err
	}
	return hash, nil
}

func (r *Repository) GetRole(ctx context.Context, userID uuid.UUID) (*Role, error) {
	var role Role
	query := `
		SELECT * FROM roles
		WHERE user_id = $1
		AND (expires_at IS NULL OR expires_at > NOW())`

	if err := r.db.GetContext(ctx, &role, query, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &role, nil
}

func (r *Repository) GetStaffProfile(ctx context.Context, userID uuid.UUID) (*StaffProfile, error) {
	var profile StaffProfile
	query := `SELECT * FROM staff_profiles WHERE user_id = $1`

	if err := r.db.GetContext(ctx, &profile, query, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrStaffProfileNotFound
		}
		return nil, err
	}
	return &profile, nil
}

func (r *Repository) CreateStaffProfile(ctx context.Context, profile *StaffProfile) error {
	query := `
		INSERT INTO staff_profiles (user_id, highest_degree, field_of_study, years_of_service, salary, salary_currency)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`

	return r.db.QueryRowxContext(ctx, query,
		profile.UserID,
		profile.HighestDegree,
		profile.FieldOfStudy,
		profile.YearsOfService,
		profile.Salary,
		profile.SalaryCurrency,
	).Scan(&profile.ID, &profile.CreatedAt, &profile.UpdatedAt)
}

func (r *Repository) UpdateStaffProfile(ctx context.Context, profile *StaffProfile) error {
	query := `
		UPDATE staff_profiles
		SET highest_degree = $2, field_of_study = $3, years_of_service = $4, salary = $5, salary_currency = $6
		WHERE user_id = $1
		RETURNING updated_at`

	err := r.db.QueryRowxContext(ctx, query,
		profile.UserID,
		profile.HighestDegree,
		profile.FieldOfStudy,
		profile.YearsOfService,
		profile.Salary,
		profile.SalaryCurrency,
	).Scan(&profile.UpdatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return ErrStaffProfileNotFound
	}
	return err
}

func (r *Repository) SetPassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	query := `UPDATE users SET password_hash = $2 WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, userID, passwordHash)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *Repository) ScopeExists(ctx context.Context, scopeType string, scopeID uuid.UUID) (bool, error) {
	var exists bool
	var query string

	switch scopeType {
	case "college":
		query = `SELECT EXISTS(SELECT 1 FROM colleges WHERE id = $1)`
	case "department":
		query = `SELECT EXISTS(SELECT 1 FROM departments WHERE id = $1)`
	case "program":
		query = `SELECT EXISTS(SELECT 1 FROM programs WHERE id = $1)`
	default:
		return false, ErrInvalidScopeID
	}

	err := r.db.GetContext(ctx, &exists, query, scopeID)
	return exists, err
}

func (r *Repository) CreateStaffUserTx(ctx context.Context, user *User, profile *StaffProfile, role *Role) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	userQuery := `
		INSERT INTO users (email, password_hash, full_name_en, full_name_ku)
		VALUES ($1, $2, $3, $4)
		RETURNING *`
	if err := tx.QueryRowxContext(ctx, userQuery, user.Email, user.PasswordHash, user.FullNameEN, user.FullNameKU).StructScan(user); err != nil {
		return err
	}

	profile.UserID = user.ID
	profileQuery := `
		INSERT INTO staff_profiles (user_id, highest_degree, field_of_study, years_of_service, salary, salary_currency)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`
	if err := tx.QueryRowxContext(ctx, profileQuery,
		profile.UserID, profile.HighestDegree, profile.FieldOfStudy,
		profile.YearsOfService, profile.Salary, profile.SalaryCurrency,
	).Scan(&profile.ID, &profile.CreatedAt, &profile.UpdatedAt); err != nil {
		return err
	}

	if role != nil {
		role.UserID = user.ID
		roleQuery := `
			INSERT INTO roles (user_id, title, permission, scope_type, scope_id, assigned_by)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id, created_at`
		if err := tx.QueryRowxContext(ctx, roleQuery,
			role.UserID, role.Title, role.Permission, role.ScopeType, role.ScopeID, role.AssignedBy,
		).Scan(&role.ID, &role.CreatedAt); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *Repository) CreateRole(ctx context.Context, role *Role) error {
	query := `
		INSERT INTO roles (user_id, title, permission, scope_type, scope_id, assigned_by, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at`

	err := r.db.QueryRowxContext(ctx, query,
		role.UserID, role.Title, role.Permission, role.ScopeType, role.ScopeID, role.AssignedBy, role.ExpiresAt,
	).Scan(&role.ID, &role.CreatedAt)

	if err != nil && strings.Contains(err.Error(), "roles_user_id_key") {
		return ErrRoleExists
	}
	return err
}

func (r *Repository) UpdateRole(ctx context.Context, role *Role) error {
	query := `
		UPDATE roles
		SET title = $2, permission = $3, scope_type = $4, scope_id = $5, assigned_by = $6, expires_at = $7
		WHERE user_id = $1
		RETURNING id, created_at`

	err := r.db.QueryRowxContext(ctx, query,
		role.UserID, role.Title, role.Permission, role.ScopeType, role.ScopeID, role.AssignedBy, role.ExpiresAt,
	).Scan(&role.ID, &role.CreatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return ErrRoleNotFound
	}
	return err
}

func (r *Repository) DeleteRole(ctx context.Context, userID uuid.UUID) error {
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
		return ErrRoleNotFound
	}
	return nil
}
