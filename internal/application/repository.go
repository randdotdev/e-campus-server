package application

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

// Application CRUD operations

func (r *Repository) Create(ctx context.Context, app *Application) error {
	query := `
		INSERT INTO applications (
			user_id, program_id, admission_year, study_type,
			date_of_birth, gender, nationality,
			personal_extra, academic, documents, status
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at, updated_at`

	return r.db.QueryRowxContext(ctx, query,
		app.UserID, app.ProgramID, app.AdmissionYear, app.StudyType,
		app.DateOfBirth, app.Gender, app.Nationality,
		app.PersonalExtra, app.Academic, app.Documents, app.Status,
	).Scan(&app.ID, &app.CreatedAt, &app.UpdatedAt)
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Application, error) {
	var app Application
	query := `SELECT * FROM applications WHERE id = $1`

	if err := r.db.GetContext(ctx, &app, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrApplicationNotFound
		}
		return nil, err
	}
	return &app, nil
}

func (r *Repository) Update(ctx context.Context, app *Application) error {
	query := `
		UPDATE applications
		SET personal_extra = $2, academic = $3, documents = $4, status = $5,
			reviewed_by = $6, reviewed_at = $7, review_notes = $8
		WHERE id = $1
		RETURNING updated_at`

	err := r.db.QueryRowxContext(ctx, query,
		app.ID, app.PersonalExtra, app.Academic, app.Documents, app.Status,
		app.ReviewedBy, app.ReviewedAt, app.ReviewNotes,
	).Scan(&app.UpdatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return ErrApplicationNotFound
	}
	return err
}

func (r *Repository) HasPendingApplication(ctx context.Context, userID, programID uuid.UUID, admissionYear int) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS(
			SELECT 1 FROM applications
			WHERE user_id = $1 AND program_id = $2 AND admission_year = $3
			AND status IN ('pending', 'needs_revision')
		)`
	err := r.db.GetContext(ctx, &exists, query, userID, programID, admissionYear)
	return exists, err
}

func (r *Repository) List(ctx context.Context, params pagination.PageParams, filters ApplicationFilters) ([]Application, bool, error) {
	query := strings.Builder{}
	args := []any{}
	argN := 1

	query.WriteString(`
		SELECT a.* FROM applications a
		JOIN programs p ON a.program_id = p.id
		JOIN departments d ON p.department_id = d.id
		WHERE 1=1`)

	if params.Cursor != "" {
		createdAt, id, err := pagination.DecodeCursor(params.Cursor)
		if err != nil {
			return nil, false, err
		}
		query.WriteString(fmt.Sprintf(" AND (a.created_at, a.id) < ($%d, $%d)", argN, argN+1))
		args = append(args, createdAt, id)
		argN += 2
	}

	if params.Query != "" {
		query.WriteString(fmt.Sprintf(`
			AND EXISTS (
				SELECT 1 FROM users u
				WHERE u.id = a.user_id
				AND (u.email ILIKE $%d OR u.full_name_en ILIKE $%d OR u.full_name_ku ILIKE $%d)
			)`, argN, argN, argN))
		args = append(args, "%"+pagination.EscapeLike(params.Query)+"%")
		argN++
	}

	if filters.ProgramID != nil {
		query.WriteString(fmt.Sprintf(" AND a.program_id = $%d", argN))
		args = append(args, *filters.ProgramID)
		argN++
	}

	if filters.DepartmentID != nil {
		query.WriteString(fmt.Sprintf(" AND p.department_id = $%d", argN))
		args = append(args, *filters.DepartmentID)
		argN++
	}

	if filters.CollegeID != nil {
		query.WriteString(fmt.Sprintf(" AND d.college_id = $%d", argN))
		args = append(args, *filters.CollegeID)
		argN++
	}

	if filters.Status != nil {
		query.WriteString(fmt.Sprintf(" AND a.status = $%d", argN))
		args = append(args, *filters.Status)
		argN++
	}

	if filters.AdmissionYear != nil {
		query.WriteString(fmt.Sprintf(" AND a.admission_year = $%d", argN))
		args = append(args, *filters.AdmissionYear)
		argN++
	}

	if filters.StudyType != nil {
		query.WriteString(fmt.Sprintf(" AND a.study_type = $%d", argN))
		args = append(args, *filters.StudyType)
		argN++
	}

	if filters.Nationality != nil {
		query.WriteString(fmt.Sprintf(" AND a.nationality = $%d", argN))
		args = append(args, *filters.Nationality)
		argN++
	}

	if filters.Gender != nil {
		query.WriteString(fmt.Sprintf(" AND a.gender = $%d", argN))
		args = append(args, *filters.Gender)
		argN++
	}

	if filters.UserID != nil {
		query.WriteString(fmt.Sprintf(" AND a.user_id = $%d", argN))
		args = append(args, *filters.UserID)
		argN++
	}

	query.WriteString(" ORDER BY a.created_at DESC, a.id DESC")
	query.WriteString(fmt.Sprintf(" LIMIT $%d", argN))
	args = append(args, params.Limit+1)

	var apps []Application
	if err := r.db.SelectContext(ctx, &apps, query.String(), args...); err != nil {
		return nil, false, err
	}

	hasMore := len(apps) > params.Limit
	if hasMore {
		apps = apps[:params.Limit]
	}

	return apps, hasMore, nil
}

func (r *Repository) ListByUser(ctx context.Context, userID uuid.UUID, params pagination.PageParams) ([]Application, bool, error) {
	query := strings.Builder{}
	args := []any{userID}
	argN := 2

	query.WriteString("SELECT * FROM applications WHERE user_id = $1")

	if params.Cursor != "" {
		createdAt, id, err := pagination.DecodeCursor(params.Cursor)
		if err != nil {
			return nil, false, err
		}
		query.WriteString(fmt.Sprintf(" AND (created_at, id) < ($%d, $%d)", argN, argN+1))
		args = append(args, createdAt, id)
		argN += 2
	}

	query.WriteString(" ORDER BY created_at DESC, id DESC")
	query.WriteString(fmt.Sprintf(" LIMIT $%d", argN))
	args = append(args, params.Limit+1)

	var apps []Application
	if err := r.db.SelectContext(ctx, &apps, query.String(), args...); err != nil {
		return nil, false, err
	}

	hasMore := len(apps) > params.Limit
	if hasMore {
		apps = apps[:params.Limit]
	}

	return apps, hasMore, nil
}

// Program lookup operations

func (r *Repository) GetProgramHierarchy(ctx context.Context, programID uuid.UUID) (*ProgramHierarchy, error) {
	var h ProgramHierarchy
	query := `
		SELECT p.id as program_id, p.department_id, d.college_id
		FROM programs p
		JOIN departments d ON p.department_id = d.id
		WHERE p.id = $1`

	if err := r.db.GetContext(ctx, &h, query, programID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProgramNotFound
		}
		return nil, err
	}
	return &h, nil
}

func (r *Repository) IsProgramActive(ctx context.Context, programID uuid.UUID) (bool, error) {
	var isActive bool
	query := `SELECT is_active FROM programs WHERE id = $1`
	if err := r.db.GetContext(ctx, &isActive, query, programID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, ErrProgramNotFound
		}
		return false, err
	}
	return isActive, nil
}

type ProgramAgeRequirements struct {
	MinAge *int
	MaxAge *int
}

func (r *Repository) GetProgramAgeRequirements(ctx context.Context, programID uuid.UUID) (*ProgramAgeRequirements, error) {
	var req ProgramAgeRequirements
	query := `SELECT min_age, max_age FROM programs WHERE id = $1`
	if err := r.db.QueryRowxContext(ctx, query, programID).Scan(&req.MinAge, &req.MaxAge); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProgramNotFound
		}
		return nil, err
	}
	return &req, nil
}
