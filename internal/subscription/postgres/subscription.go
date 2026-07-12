// Package postgres holds the SQL adapter for the subscription domain.
package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/randdotdev/e-campus-server/internal/subscription"
)

// Repository is the SQL adapter for the subscription context.
type Repository struct {
	db *sqlx.DB
}

var _ subscription.Repository = (*Repository)(nil)

// NewRepository wires the subscription adapter.
func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Get(ctx context.Context) (*subscription.Subscription, error) {
	var sub subscription.Subscription
	if err := r.db.GetContext(ctx, &sub, `SELECT * FROM subscription LIMIT 1`); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, subscription.ErrSubscriptionNotFound
		}
		return nil, err
	}
	return &sub, nil
}

func (r *Repository) GetTierLimits(ctx context.Context, tier subscription.Tier) (*subscription.TierLimits, error) {
	var tl subscription.TierLimits
	if err := r.db.GetContext(ctx, &tl, `SELECT * FROM tier_limits WHERE tier = $1`, tier); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, subscription.ErrTierNotFound
		}
		return nil, err
	}
	return &tl, nil
}

func (r *Repository) GetAllTierLimits(ctx context.Context) ([]subscription.TierLimits, error) {
	var tiers []subscription.TierLimits
	if err := r.db.SelectContext(ctx, &tiers, `SELECT * FROM tier_limits ORDER BY tier`); err != nil {
		return nil, err
	}
	return tiers, nil
}

func (r *Repository) UpdateTierLimits(ctx context.Context, tl *subscription.TierLimits) error {
	query := `
		UPDATE tier_limits
		SET max_colleges = $2, max_departments_per_college = $3, max_programs_per_department = $4,
		    max_students_per_program = $5, max_applications_per_user = $6, max_staff_users = $7,
		    updated_at = NOW()
		WHERE tier = $1
		RETURNING updated_at`
	return r.db.QueryRowxContext(ctx, query,
		tl.Tier, tl.MaxColleges, tl.MaxDepartmentsPerCollege, tl.MaxProgramsPerDepartment,
		tl.MaxStudentsPerProgram, tl.MaxApplicationsPerUser, tl.MaxStaffUsers,
	).Scan(&tl.UpdatedAt)
}

func (r *Repository) GetHistory(ctx context.Context, limit int) ([]subscription.History, error) {
	var history []subscription.History
	if err := r.db.SelectContext(ctx, &history, `SELECT * FROM subscription_history ORDER BY changed_at DESC LIMIT $1`, limit); err != nil {
		return nil, err
	}
	return history, nil
}

// UpdateWithHistory is an optimistic compare-and-swap keyed on expectedVersion.
// It updates the subscription and appends a history row in one transaction,
// returning the new version; a version mismatch (zero rows updated) is
// ErrConflict.
func (r *Repository) UpdateWithHistory(ctx context.Context, sub *subscription.Subscription, expectedVersion int64, reason string, changedBy *uuid.UUID) (int64, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	sub.UpdatedBy = changedBy
	const updateQuery = `
		UPDATE subscription
		SET tier = $1, max_colleges_override = $2, max_departments_override = $3,
		    max_programs_override = $4, max_students_override = $5,
		    max_applications_override = $6, max_staff_override = $7,
		    max_storage_override = $8, max_file_size_override = $9,
		    expires_at = $10, updated_by = $11, updated_at = NOW(),
		    version = version + 1
		WHERE id = $12 AND version = $13
		RETURNING version`
	var newVersion int64
	err = tx.QueryRowxContext(ctx, updateQuery,
		sub.Tier, sub.MaxCollegesOverride, sub.MaxDepartmentsOverride, sub.MaxProgramsOverride,
		sub.MaxStudentsOverride, sub.MaxApplicationsOverride, sub.MaxStaffOverride,
		sub.MaxStorageOverride, sub.MaxFileSizeOverride,
		sub.ExpiresAt, sub.UpdatedBy, sub.ID, expectedVersion,
	).Scan(&newVersion)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, subscription.ErrConflict
	}
	if err != nil {
		return 0, err
	}

	const historyQuery = `
		INSERT INTO subscription_history
		(tier, max_colleges_override, max_departments_override, max_programs_override,
		 max_students_override, max_applications_override, max_staff_override,
		 max_storage_override, max_file_size_override,
		 expires_at, changed_by, change_reason)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`
	if _, err := tx.ExecContext(ctx, historyQuery,
		sub.Tier, sub.MaxCollegesOverride, sub.MaxDepartmentsOverride, sub.MaxProgramsOverride,
		sub.MaxStudentsOverride, sub.MaxApplicationsOverride, sub.MaxStaffOverride,
		sub.MaxStorageOverride, sub.MaxFileSizeOverride,
		sub.ExpiresAt, changedBy, reason,
	); err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return newVersion, nil
}
