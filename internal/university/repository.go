package university

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/ranjdotdev/e-campus-server/internal/pagination"
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// College operations

func (r *Repository) CreateCollege(ctx context.Context, college *College) error {
	query := `
		INSERT INTO colleges (name_en, name_ku, code, description)
		VALUES ($1, $2, $3, $4)
		RETURNING id, is_active, created_at, updated_at`

	return r.db.QueryRowxContext(ctx, query,
		college.NameEN, college.NameKU, college.Code, college.Description,
	).Scan(&college.ID, &college.IsActive, &college.CreatedAt, &college.UpdatedAt)
}

func (r *Repository) GetCollege(ctx context.Context, id uuid.UUID) (*College, error) {
	var college College
	query := `SELECT * FROM colleges WHERE id = $1`

	if err := r.db.GetContext(ctx, &college, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCollegeNotFound
		}
		return nil, err
	}
	return &college, nil
}

func (r *Repository) ListColleges(ctx context.Context, params pagination.PageParams, filters CollegeFilters) ([]College, bool, error) {
	query := strings.Builder{}
	args := []any{}
	argN := 1

	query.WriteString("SELECT * FROM colleges WHERE 1=1")

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
		query.WriteString(fmt.Sprintf(" AND (name_en ILIKE $%d OR name_ku ILIKE $%d OR code ILIKE $%d)", argN, argN, argN))
		args = append(args, "%"+pagination.EscapeLike(params.Query)+"%")
		argN++
	}

	if filters.IsActive != nil {
		query.WriteString(fmt.Sprintf(" AND is_active = $%d", argN))
		args = append(args, *filters.IsActive)
		argN++
	}

	query.WriteString(" ORDER BY created_at DESC, id DESC")
	query.WriteString(fmt.Sprintf(" LIMIT $%d", argN))
	args = append(args, params.Limit+1)

	var colleges []College
	if err := r.db.SelectContext(ctx, &colleges, query.String(), args...); err != nil {
		return nil, false, err
	}

	hasMore := len(colleges) > params.Limit
	if hasMore {
		colleges = colleges[:params.Limit]
	}

	return colleges, hasMore, nil
}

func (r *Repository) UpdateCollege(ctx context.Context, college *College) error {
	query := `
		UPDATE colleges
		SET name_en = $2, name_ku = $3, code = $4, description = $5, is_active = $6
		WHERE id = $1
		RETURNING updated_at`

	err := r.db.QueryRowxContext(ctx, query,
		college.ID, college.NameEN, college.NameKU, college.Code, college.Description, college.IsActive,
	).Scan(&college.UpdatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return ErrCollegeNotFound
	}
	return err
}

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

// Department operations

func (r *Repository) CreateDepartment(ctx context.Context, dept *Department) error {
	query := `
		INSERT INTO departments (college_id, name_en, name_ku, code, description)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, is_active, created_at, updated_at`

	return r.db.QueryRowxContext(ctx, query,
		dept.CollegeID, dept.NameEN, dept.NameKU, dept.Code, dept.Description,
	).Scan(&dept.ID, &dept.IsActive, &dept.CreatedAt, &dept.UpdatedAt)
}

func (r *Repository) GetDepartment(ctx context.Context, id uuid.UUID) (*Department, error) {
	var dept Department
	query := `SELECT * FROM departments WHERE id = $1`

	if err := r.db.GetContext(ctx, &dept, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDepartmentNotFound
		}
		return nil, err
	}
	return &dept, nil
}

func (r *Repository) ListDepartments(ctx context.Context, params pagination.PageParams, filters DepartmentFilters) ([]Department, bool, error) {
	query := strings.Builder{}
	args := []any{}
	argN := 1

	query.WriteString("SELECT * FROM departments WHERE 1=1")

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
		query.WriteString(fmt.Sprintf(" AND (name_en ILIKE $%d OR name_ku ILIKE $%d OR code ILIKE $%d)", argN, argN, argN))
		args = append(args, "%"+pagination.EscapeLike(params.Query)+"%")
		argN++
	}

	if filters.CollegeID != nil {
		query.WriteString(fmt.Sprintf(" AND college_id = $%d", argN))
		args = append(args, *filters.CollegeID)
		argN++
	}

	if filters.IsActive != nil {
		query.WriteString(fmt.Sprintf(" AND is_active = $%d", argN))
		args = append(args, *filters.IsActive)
		argN++
	}

	query.WriteString(" ORDER BY created_at DESC, id DESC")
	query.WriteString(fmt.Sprintf(" LIMIT $%d", argN))
	args = append(args, params.Limit+1)

	var depts []Department
	if err := r.db.SelectContext(ctx, &depts, query.String(), args...); err != nil {
		return nil, false, err
	}

	hasMore := len(depts) > params.Limit
	if hasMore {
		depts = depts[:params.Limit]
	}

	return depts, hasMore, nil
}

func (r *Repository) UpdateDepartment(ctx context.Context, dept *Department) error {
	query := `
		UPDATE departments
		SET name_en = $2, name_ku = $3, code = $4, description = $5, is_active = $6
		WHERE id = $1
		RETURNING updated_at`

	err := r.db.QueryRowxContext(ctx, query,
		dept.ID, dept.NameEN, dept.NameKU, dept.Code, dept.Description, dept.IsActive,
	).Scan(&dept.UpdatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return ErrDepartmentNotFound
	}
	return err
}

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

// Program operations

func (r *Repository) CreateProgram(ctx context.Context, program *Program) error {
	query := `
		INSERT INTO programs (department_id, name_en, name_ku, code, degree_type, duration_years, total_ects, min_age, max_age, description)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, is_active, created_at, updated_at`

	return r.db.QueryRowxContext(ctx, query,
		program.DepartmentID, program.NameEN, program.NameKU, program.Code,
		program.DegreeType, program.DurationYears, program.TotalECTS, program.MinAge, program.MaxAge, program.Description,
	).Scan(&program.ID, &program.IsActive, &program.CreatedAt, &program.UpdatedAt)
}

func (r *Repository) GetProgram(ctx context.Context, id uuid.UUID) (*Program, error) {
	var program Program
	query := `SELECT * FROM programs WHERE id = $1`

	if err := r.db.GetContext(ctx, &program, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProgramNotFound
		}
		return nil, err
	}
	return &program, nil
}

func (r *Repository) ListPrograms(ctx context.Context, params pagination.PageParams, filters ProgramFilters) ([]Program, bool, error) {
	query := strings.Builder{}
	args := []any{}
	argN := 1

	query.WriteString("SELECT * FROM programs WHERE 1=1")

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
		query.WriteString(fmt.Sprintf(" AND (name_en ILIKE $%d OR name_ku ILIKE $%d OR code ILIKE $%d)", argN, argN, argN))
		args = append(args, "%"+pagination.EscapeLike(params.Query)+"%")
		argN++
	}

	if filters.DepartmentID != nil {
		query.WriteString(fmt.Sprintf(" AND department_id = $%d", argN))
		args = append(args, *filters.DepartmentID)
		argN++
	}

	if filters.DegreeType != nil {
		query.WriteString(fmt.Sprintf(" AND degree_type = $%d", argN))
		args = append(args, *filters.DegreeType)
		argN++
	}

	if filters.IsActive != nil {
		query.WriteString(fmt.Sprintf(" AND is_active = $%d", argN))
		args = append(args, *filters.IsActive)
		argN++
	}

	query.WriteString(" ORDER BY created_at DESC, id DESC")
	query.WriteString(fmt.Sprintf(" LIMIT $%d", argN))
	args = append(args, params.Limit+1)

	var programs []Program
	if err := r.db.SelectContext(ctx, &programs, query.String(), args...); err != nil {
		return nil, false, err
	}

	hasMore := len(programs) > params.Limit
	if hasMore {
		programs = programs[:params.Limit]
	}

	return programs, hasMore, nil
}

func (r *Repository) UpdateProgram(ctx context.Context, program *Program) error {
	query := `
		UPDATE programs
		SET name_en = $2, name_ku = $3, code = $4, degree_type = $5, duration_years = $6, total_ects = $7, min_age = $8, max_age = $9, description = $10, is_active = $11
		WHERE id = $1
		RETURNING updated_at`

	err := r.db.QueryRowxContext(ctx, query,
		program.ID, program.NameEN, program.NameKU, program.Code,
		program.DegreeType, program.DurationYears, program.TotalECTS, program.MinAge, program.MaxAge, program.Description, program.IsActive,
	).Scan(&program.UpdatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return ErrProgramNotFound
	}
	return err
}

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

// Count operations

func (r *Repository) CountColleges(ctx context.Context) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM colleges`
	err := r.db.GetContext(ctx, &count, query)
	return count, err
}

func (r *Repository) CountDepartmentsByCollege(ctx context.Context, collegeID uuid.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM departments WHERE college_id = $1`
	err := r.db.GetContext(ctx, &count, query, collegeID)
	return count, err
}

func (r *Repository) CountProgramsByDepartment(ctx context.Context, departmentID uuid.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM programs WHERE department_id = $1`
	err := r.db.GetContext(ctx, &count, query, departmentID)
	return count, err
}
