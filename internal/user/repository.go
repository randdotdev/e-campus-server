package user

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/ranjdotdev/e-campus-server/internal/auth"
)

var (
	ErrUserNotFound         = errors.New("user not found")
	ErrStaffProfileNotFound = errors.New("staff profile not found")
	ErrEmailExists          = errors.New("email already exists")
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

func (r *Repository) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]auth.RoleData, error) {
	var roles []Role
	query := `
		SELECT * FROM roles
		WHERE user_id = $1
		AND (expires_at IS NULL OR expires_at > NOW())`

	if err := r.db.SelectContext(ctx, &roles, query, userID); err != nil {
		return nil, err
	}
	return toRoleDataSlice(roles), nil
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

func toRoleDataSlice(roles []Role) []auth.RoleData {
	result := make([]auth.RoleData, len(roles))
	for i, r := range roles {
		result[i] = auth.RoleData{
			ID:         r.ID,
			Title:      r.Title,
			Permission: r.Permission,
			ScopeType:  r.ScopeType,
			ScopeID:    r.ScopeID,
		}
	}
	return result
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

func (r *Repository) List(ctx context.Context, limit, offset int) ([]User, int, error) {
	var users []User
	var total int

	countQuery := `SELECT COUNT(*) FROM users`
	if err := r.db.GetContext(ctx, &total, countQuery); err != nil {
		return nil, 0, err
	}

	query := `SELECT * FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	if err := r.db.SelectContext(ctx, &users, query, limit, offset); err != nil {
		return nil, 0, err
	}

	return users, total, nil
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

func (r *Repository) GetRoles(ctx context.Context, userID uuid.UUID) ([]Role, error) {
	var roles []Role
	query := `
		SELECT * FROM roles
		WHERE user_id = $1
		AND (expires_at IS NULL OR expires_at > NOW())`

	if err := r.db.SelectContext(ctx, &roles, query, userID); err != nil {
		return nil, err
	}
	return roles, nil
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
