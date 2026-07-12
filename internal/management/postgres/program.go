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

// ── Programs ─────────────────────────────────────────────────────────────────

// CreateProgram inserts a program.
func (r *Repository) CreateProgram(ctx context.Context, program *management.Program) error {
	const query = `
		INSERT INTO programs (department_id, name_en, name_local, code, degree_type, duration_years, total_credits, min_age, max_age, description)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, is_active, created_at, updated_at, version`

	return r.db.QueryRowxContext(ctx, query,
		program.DepartmentID, program.NameEN, program.NameLocal, program.Code,
		program.DegreeType, program.DurationYears, program.TotalCredits, program.MinAge, program.MaxAge, program.Description,
	).Scan(&program.ID, &program.IsActive, &program.CreatedAt, &program.UpdatedAt, &program.Version)
}

// GetProgram fetches one program.
func (r *Repository) GetProgram(ctx context.Context, id uuid.UUID) (*management.Program, error) {
	var program management.Program
	const query = `SELECT * FROM programs WHERE id = $1`

	if err := r.db.GetContext(ctx, &program, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, management.ErrProgramNotFound
		}
		return nil, err
	}
	return &program, nil
}

// ListPrograms pages through programs matching the filter, newest first.
func (r *Repository) ListPrograms(ctx context.Context, params pagination.PageParams, filters management.ProgramFilter) ([]management.Program, bool, error) {
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
	if filters.DepartmentID != nil {
		conditions = append(conditions, fmt.Sprintf("department_id = $%d", argN))
		args = append(args, *filters.DepartmentID)
		argN++
	}
	if filters.DegreeType != nil {
		conditions = append(conditions, fmt.Sprintf("degree_type = $%d", argN))
		args = append(args, *filters.DegreeType)
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
	query := fmt.Sprintf("SELECT * FROM programs %s ORDER BY created_at DESC, id DESC LIMIT $%d", where, argN)
	args = append(args, params.Limit+1)

	var programs []management.Program
	if err := r.db.SelectContext(ctx, &programs, query, args...); err != nil {
		return nil, false, err
	}

	hasMore := len(programs) > params.Limit
	if hasMore {
		programs = programs[:params.Limit]
	}
	return programs, hasMore, nil
}

// UpdateProgram is an optimistic compare-and-swap keyed on version.
func (r *Repository) UpdateProgram(ctx context.Context, program *management.Program, expectedVersion int64) (int64, error) {
	const query = `
		UPDATE programs
		   SET name_en = $2, name_local = $3, code = $4, degree_type = $5, duration_years = $6,
		       total_credits = $7, min_age = $8, max_age = $9, description = $10, is_active = $11,
		       version = version + 1, updated_at = NOW()
		 WHERE id = $1 AND version = $12
		RETURNING version`

	var newVersion int64
	err := r.db.QueryRowxContext(ctx, query,
		program.ID, program.NameEN, program.NameLocal, program.Code,
		program.DegreeType, program.DurationYears, program.TotalCredits, program.MinAge, program.MaxAge, program.Description, program.IsActive,
		expectedVersion,
	).Scan(&newVersion)
	if errors.Is(err, sql.ErrNoRows) {
		if exists, exErr := r.ProgramExists(ctx, program.ID); exErr == nil && !exists {
			return 0, management.ErrProgramNotFound
		}
		return 0, management.ErrConflict
	}
	if err != nil {
		return 0, err
	}
	return newVersion, nil
}

// ProgramCodeExists reports whether another program in the department
// already uses the code.
func (r *Repository) ProgramCodeExists(ctx context.Context, departmentID uuid.UUID, code string, excludeID *uuid.UUID) (bool, error) {
	var exists bool
	var query string
	var args []any
	if excludeID != nil {
		query = `SELECT EXISTS(SELECT 1 FROM programs WHERE department_id = $1 AND code = $2 AND id != $3)`
		args = []any{departmentID, code, *excludeID}
	} else {
		query = `SELECT EXISTS(SELECT 1 FROM programs WHERE department_id = $1 AND code = $2)`
		args = []any{departmentID, code}
	}
	err := r.db.GetContext(ctx, &exists, query, args...)
	return exists, err
}

// CountProgramsByDepartment counts a department's programs for the
// subscription limit check.
func (r *Repository) CountProgramsByDepartment(ctx context.Context, departmentID uuid.UUID) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM programs WHERE department_id = $1`, departmentID)
	return count, err
}

// ── Cross-context reads (student.ProgramProvider) ────────────────────────────
//
// Consumed by the student context. The interface assertion lives on student's
// side once it migrates; structural typing means management need not import it.

// ProgramExists reports whether the program exists. It satisfies
// management.StudentProgramProvider.
func (r *Repository) ProgramExists(ctx context.Context, id uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists, `SELECT EXISTS(SELECT 1 FROM programs WHERE id = $1)`, id)
	return exists, err
}

// GetProgramTotalCredits returns the program's total required credits. It
// satisfies management.StudentProgramProvider.
func (r *Repository) GetProgramTotalCredits(ctx context.Context, id uuid.UUID) (int, error) {
	var credits int
	err := r.db.GetContext(ctx, &credits, `SELECT total_credits FROM programs WHERE id = $1`, id)
	return credits, err
}
