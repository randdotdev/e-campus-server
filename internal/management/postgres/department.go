package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/management"
	"github.com/randdotdev/e-campus-server/internal/shared/pagination"
)

// ── Departments ──────────────────────────────────────────────────────────────

// CreateDepartment inserts a department.
func (r *Repository) CreateDepartment(ctx context.Context, dept *management.Department) error {
	const query = `
		INSERT INTO departments (college_id, name_en, name_local, code, description, about, founded, phone, email, logo_url)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, is_active, created_at, updated_at, version`

	return r.db.QueryRowxContext(ctx, query,
		dept.CollegeID, dept.NameEN, dept.NameLocal, dept.Code, dept.Description,
		dept.About, dept.Founded, dept.Phone, dept.Email, dept.LogoURL,
	).Scan(&dept.ID, &dept.IsActive, &dept.CreatedAt, &dept.UpdatedAt, &dept.Version)
}

// GetDepartment fetches one department.
func (r *Repository) GetDepartment(ctx context.Context, id uuid.UUID) (*management.Department, error) {
	var dept management.Department
	const query = `SELECT * FROM departments WHERE id = $1`

	if err := r.db.GetContext(ctx, &dept, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, management.ErrDepartmentNotFound
		}
		return nil, err
	}
	return &dept, nil
}

// ListDepartments pages through departments matching the filter, newest
// first.
func (r *Repository) ListDepartments(ctx context.Context, params pagination.PageParams, filters management.DepartmentFilter) ([]management.Department, bool, error) {
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
		conditions = append(conditions, fmt.Sprintf("(name_en ILIKE $%d OR name_local ILIKE $%d OR code ILIKE $%d)", argN, argN, argN))
		args = append(args, "%"+pagination.EscapeLike(params.Query)+"%")
		argN++
	}
	if filters.CollegeID != nil {
		conditions = append(conditions, fmt.Sprintf("college_id = $%d", argN))
		args = append(args, *filters.CollegeID)
		argN++
	}
	if filters.IsActive != nil {
		conditions = append(conditions, fmt.Sprintf("is_active = $%d", argN))
		args = append(args, *filters.IsActive)
		argN++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}
	query := fmt.Sprintf("SELECT * FROM departments %s ORDER BY created_at DESC, id DESC LIMIT $%d", where, argN)
	args = append(args, params.Limit+1)

	var depts []management.Department
	if err := r.db.SelectContext(ctx, &depts, query, args...); err != nil {
		return nil, false, err
	}

	hasMore := len(depts) > params.Limit
	if hasMore {
		depts = depts[:params.Limit]
	}
	return depts, hasMore, nil
}

// UpdateDepartment is an optimistic compare-and-swap keyed on version.
func (r *Repository) UpdateDepartment(ctx context.Context, dept *management.Department, expectedVersion int64) (int64, error) {
	const query = `
		UPDATE departments
		   SET name_en = $2, name_local = $3, code = $4, description = $5, is_active = $6,
		       about = $7, founded = $8, phone = $9, email = $10, logo_url = $11,
		       version = version + 1, updated_at = NOW()
		 WHERE id = $1 AND version = $12
		RETURNING version`

	var newVersion int64
	err := r.db.QueryRowxContext(ctx, query,
		dept.ID, dept.NameEN, dept.NameLocal, dept.Code, dept.Description, dept.IsActive,
		dept.About, dept.Founded, dept.Phone, dept.Email, dept.LogoURL, expectedVersion,
	).Scan(&newVersion)
	if errors.Is(err, sql.ErrNoRows) {
		if exists, exErr := r.departmentExists(ctx, dept.ID); exErr == nil && !exists {
			return 0, management.ErrDepartmentNotFound
		}
		return 0, management.ErrConflict
	}
	if err != nil {
		return 0, err
	}
	return newVersion, nil
}

// DepartmentCodeExists reports whether another department in the college
// already uses the code.
func (r *Repository) DepartmentCodeExists(ctx context.Context, collegeID uuid.UUID, code string, excludeID *uuid.UUID) (bool, error) {
	var exists bool
	var query string
	var args []any
	if excludeID != nil {
		query = `SELECT EXISTS(SELECT 1 FROM departments WHERE college_id = $1 AND code = $2 AND id != $3)`
		args = []any{collegeID, code, *excludeID}
	} else {
		query = `SELECT EXISTS(SELECT 1 FROM departments WHERE college_id = $1 AND code = $2)`
		args = []any{collegeID, code}
	}
	err := r.db.GetContext(ctx, &exists, query, args...)
	return exists, err
}

// CountDepartmentsByCollege counts a college's departments for the
// subscription limit check.
func (r *Repository) CountDepartmentsByCollege(ctx context.Context, collegeID uuid.UUID) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM departments WHERE college_id = $1`, collegeID)
	return count, err
}

func (r *Repository) departmentExists(ctx context.Context, id uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists, `SELECT EXISTS(SELECT 1 FROM departments WHERE id = $1)`, id)
	return exists, err
}
