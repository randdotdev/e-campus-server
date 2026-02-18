package subscription

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var (
	ErrSubscriptionNotFound = errors.New("subscription not found")
	ErrTierNotFound         = errors.New("tier not found")
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// Tier limits operations

func (r *Repository) GetTierLimits(ctx context.Context, tier string) (*TierLimits, error) {
	var tl TierLimits
	query := `SELECT * FROM tier_limits WHERE tier = $1`

	if err := r.db.GetContext(ctx, &tl, query, tier); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTierNotFound
		}
		return nil, err
	}
	return &tl, nil
}

func (r *Repository) GetAllTierLimits(ctx context.Context) ([]TierLimits, error) {
	var tiers []TierLimits
	query := `SELECT * FROM tier_limits ORDER BY tier`

	if err := r.db.SelectContext(ctx, &tiers, query); err != nil {
		return nil, err
	}
	return tiers, nil
}

func (r *Repository) UpdateTierLimits(ctx context.Context, tl *TierLimits) error {
	query := `
		UPDATE tier_limits
		SET max_colleges = $2, max_departments_per_college = $3, max_programs_per_department = $4,
		    max_students_per_program = $5, max_applications_per_user = $6, max_staff_users = $7,
		    updated_at = NOW()
		WHERE tier = $1
		RETURNING updated_at`

	return r.db.QueryRowxContext(ctx, query,
		tl.Tier,
		tl.MaxColleges,
		tl.MaxDepartmentsPerCollege,
		tl.MaxProgramsPerDepartment,
		tl.MaxStudentsPerProgram,
		tl.MaxApplicationsPerUser,
		tl.MaxStaffUsers,
	).Scan(&tl.UpdatedAt)
}

// Subscription operations

func (r *Repository) Get(ctx context.Context) (*Subscription, error) {
	var sub Subscription
	query := `SELECT * FROM subscription LIMIT 1`

	if err := r.db.GetContext(ctx, &sub, query); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSubscriptionNotFound
		}
		return nil, err
	}
	return &sub, nil
}

func (r *Repository) Update(ctx context.Context, sub *Subscription) error {
	query := `
		UPDATE subscription
		SET tier = $1, max_colleges_override = $2, max_departments_override = $3,
		    max_programs_override = $4, max_students_override = $5,
		    max_applications_override = $6, max_staff_override = $7,
		    expires_at = $8, updated_by = $9, updated_at = NOW()
		WHERE id = $10
		RETURNING updated_at`

	return r.db.QueryRowxContext(ctx, query,
		sub.Tier,
		sub.MaxCollegesOverride,
		sub.MaxDepartmentsOverride,
		sub.MaxProgramsOverride,
		sub.MaxStudentsOverride,
		sub.MaxApplicationsOverride,
		sub.MaxStaffOverride,
		sub.ExpiresAt,
		sub.UpdatedBy,
		sub.ID,
	).Scan(&sub.UpdatedAt)
}

// History operations

func (r *Repository) AddHistory(ctx context.Context, sub *Subscription, reason string, changedBy *uuid.UUID) error {
	query := `
		INSERT INTO subscription_history
		(tier, max_colleges_override, max_departments_override, max_programs_override,
		 max_students_override, max_applications_override, max_staff_override,
		 expires_at, changed_by, change_reason)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := r.db.ExecContext(ctx, query,
		sub.Tier,
		sub.MaxCollegesOverride,
		sub.MaxDepartmentsOverride,
		sub.MaxProgramsOverride,
		sub.MaxStudentsOverride,
		sub.MaxApplicationsOverride,
		sub.MaxStaffOverride,
		sub.ExpiresAt,
		changedBy,
		reason,
	)
	return err
}

func (r *Repository) GetHistory(ctx context.Context, limit int) ([]History, error) {
	var history []History
	query := `SELECT * FROM subscription_history ORDER BY changed_at DESC LIMIT $1`

	if err := r.db.SelectContext(ctx, &history, query, limit); err != nil {
		return nil, err
	}
	return history, nil
}

// UpdateWithHistory updates subscription and records history in a transaction.
func (r *Repository) UpdateWithHistory(ctx context.Context, sub *Subscription, reason string, changedBy *uuid.UUID) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// Update subscription
	sub.UpdatedBy = changedBy
	updateQuery := `
		UPDATE subscription
		SET tier = $1, max_colleges_override = $2, max_departments_override = $3,
		    max_programs_override = $4, max_students_override = $5,
		    max_applications_override = $6, max_staff_override = $7,
		    expires_at = $8, updated_by = $9, updated_at = NOW()
		WHERE id = $10
		RETURNING updated_at`

	if err := tx.QueryRowxContext(ctx, updateQuery,
		sub.Tier,
		sub.MaxCollegesOverride,
		sub.MaxDepartmentsOverride,
		sub.MaxProgramsOverride,
		sub.MaxStudentsOverride,
		sub.MaxApplicationsOverride,
		sub.MaxStaffOverride,
		sub.ExpiresAt,
		sub.UpdatedBy,
		sub.ID,
	).Scan(&sub.UpdatedAt); err != nil {
		return err
	}

	// Add history
	historyQuery := `
		INSERT INTO subscription_history
		(tier, max_colleges_override, max_departments_override, max_programs_override,
		 max_students_override, max_applications_override, max_staff_override,
		 expires_at, changed_by, change_reason)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	if _, err := tx.ExecContext(ctx, historyQuery,
		sub.Tier,
		sub.MaxCollegesOverride,
		sub.MaxDepartmentsOverride,
		sub.MaxProgramsOverride,
		sub.MaxStudentsOverride,
		sub.MaxApplicationsOverride,
		sub.MaxStaffOverride,
		sub.ExpiresAt,
		changedBy,
		reason,
	); err != nil {
		return err
	}

	return tx.Commit()
}
