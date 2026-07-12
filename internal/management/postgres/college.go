// Package postgres holds the SQL adapters for the management domain.
package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/randdotdev/e-campus-server/internal/management"
	"github.com/randdotdev/e-campus-server/internal/shared/pagination"
)

// Repository backs the university-structure ports (colleges, departments,
// programs) over one database. The structure entities share a hierarchy and a
// connection, so a single adapter satisfies all three consumer ports.
type Repository struct {
	db *sqlx.DB
}

var (
	_ management.CollegeRepository    = (*Repository)(nil)
	_ management.DepartmentRepository = (*Repository)(nil)
	_ management.ProgramRepository    = (*Repository)(nil)
)

// NewRepository wires the structure adapter.
func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// ── Colleges ─────────────────────────────────────────────────────────────────

// CreateCollege inserts a college.
func (r *Repository) CreateCollege(ctx context.Context, college *management.College) error {
	const query = `
		INSERT INTO colleges (name_en, name_local, code, description, about, founded, phone, email, logo_url)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, is_active, created_at, updated_at, version`

	return r.db.QueryRowxContext(ctx, query,
		college.NameEN, college.NameLocal, college.Code, college.Description,
		college.About, college.Founded, college.Phone, college.Email, college.LogoURL,
	).Scan(&college.ID, &college.IsActive, &college.CreatedAt, &college.UpdatedAt, &college.Version)
}

// GetCollege fetches one college.
func (r *Repository) GetCollege(ctx context.Context, id uuid.UUID) (*management.College, error) {
	var college management.College
	const query = `SELECT * FROM colleges WHERE id = $1`

	if err := r.db.GetContext(ctx, &college, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, management.ErrCollegeNotFound
		}
		return nil, err
	}
	return &college, nil
}

// ListColleges pages through colleges matching the filter, newest first.
func (r *Repository) ListColleges(ctx context.Context, params pagination.PageParams, filters management.CollegeFilter) ([]management.College, bool, error) {
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
	if filters.IsActive != nil {
		conditions = append(conditions, fmt.Sprintf("is_active = $%d", argN))
		args = append(args, *filters.IsActive)
		argN++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}
	query := fmt.Sprintf("SELECT * FROM colleges %s ORDER BY created_at DESC, id DESC LIMIT $%d", where, argN)
	args = append(args, params.Limit+1)

	var colleges []management.College
	if err := r.db.SelectContext(ctx, &colleges, query, args...); err != nil {
		return nil, false, err
	}

	hasMore := len(colleges) > params.Limit
	if hasMore {
		colleges = colleges[:params.Limit]
	}
	return colleges, hasMore, nil
}

// UpdateCollege is an optimistic compare-and-swap: the WHERE version guard makes
// a stale writer affect zero rows, surfacing as management.ErrConflict.
func (r *Repository) UpdateCollege(ctx context.Context, college *management.College, expectedVersion int64) (int64, error) {
	const query = `
		UPDATE colleges
		   SET name_en = $2, name_local = $3, code = $4, description = $5, is_active = $6,
		       about = $7, founded = $8, phone = $9, email = $10, logo_url = $11,
		       version = version + 1, updated_at = NOW()
		 WHERE id = $1 AND version = $12
		RETURNING version`

	var newVersion int64
	err := r.db.QueryRowxContext(ctx, query,
		college.ID, college.NameEN, college.NameLocal, college.Code, college.Description, college.IsActive,
		college.About, college.Founded, college.Phone, college.Email, college.LogoURL, expectedVersion,
	).Scan(&newVersion)
	if errors.Is(err, sql.ErrNoRows) {
		if exists, exErr := r.collegeExists(ctx, college.ID); exErr == nil && !exists {
			return 0, management.ErrCollegeNotFound
		}
		return 0, management.ErrConflict
	}
	if err != nil {
		return 0, err
	}
	return newVersion, nil
}

// CollegeCodeExists reports whether another college already uses the code.
func (r *Repository) CollegeCodeExists(ctx context.Context, code string, excludeID *uuid.UUID) (bool, error) {
	var exists bool
	var query string
	var args []any
	if excludeID != nil {
		query = `SELECT EXISTS(SELECT 1 FROM colleges WHERE code = $1 AND id != $2)`
		args = []any{code, *excludeID}
	} else {
		query = `SELECT EXISTS(SELECT 1 FROM colleges WHERE code = $1)`
		args = []any{code}
	}
	err := r.db.GetContext(ctx, &exists, query, args...)
	return exists, err
}

// CountColleges counts all colleges for the subscription limit check.
func (r *Repository) CountColleges(ctx context.Context) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM colleges`)
	return count, err
}

func (r *Repository) collegeExists(ctx context.Context, id uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists, `SELECT EXISTS(SELECT 1 FROM colleges WHERE id = $1)`, id)
	return exists, err
}
