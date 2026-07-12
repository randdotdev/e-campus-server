package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/randdotdev/e-campus-server/internal/management"
	"github.com/randdotdev/e-campus-server/internal/shared/pagination"
)

const applicationDetailSelect = `
	SELECT a.*,
		p.name_en AS program_name_en, p.name_local AS program_name_local,
		d.name_en AS department_name_en, d.name_local AS department_name_local,
		c.name_en AS college_name_en, c.name_local AS college_name_local,
		u.full_name_en AS applicant_name_en, u.full_name_local AS applicant_name_local,
		u.email AS applicant_email, u.avatar_url AS applicant_avatar_url
	FROM applications a
	JOIN programs p ON a.program_id = p.id
	JOIN departments d ON p.department_id = d.id
	JOIN colleges c ON d.college_id = c.id
	LEFT JOIN users u ON a.user_id = u.id`

// ApplicationRepository is the SQL adapter for admission applications.
type ApplicationRepository struct {
	db *sqlx.DB
}

// NewApplicationRepository wires the application adapter.
func NewApplicationRepository(db *sqlx.DB) *ApplicationRepository {
	return &ApplicationRepository{db: db}
}

var _ management.ApplicationRepository = (*ApplicationRepository)(nil)

// CreateApplication inserts an application. The partial unique index on
// (user, program, year) over pending/needs_revision rows is the duplicate
// guard.
func (r *ApplicationRepository) CreateApplication(ctx context.Context, app *management.Application) error {
	err := r.db.QueryRowxContext(ctx, `
		INSERT INTO applications (
			user_id, program_id, admission_year, shift, tuition,
			date_of_birth, gender, nationality,
			personal_extra, academic, documents, status
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at, updated_at`,
		app.UserID, app.ProgramID, app.AdmissionYear, app.Shift, app.Tuition,
		app.DateOfBirth, app.Gender, app.Nationality,
		app.PersonalExtra, app.Academic, app.Documents, app.Status,
	).Scan(&app.ID, &app.CreatedAt, &app.UpdatedAt)
	if isUniqueViolation(err) {
		return management.ErrDuplicateApplication
	}
	if isForeignKeyViolation(err) {
		return management.ErrProgramNotFound
	}
	return err
}

// GetApplication fetches one application with its display joins.
func (r *ApplicationRepository) GetApplication(ctx context.Context, id uuid.UUID) (*management.ApplicationDetail, error) {
	var app management.ApplicationDetail
	if err := r.db.GetContext(ctx, &app, applicationDetailSelect+` WHERE a.id = $1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, management.ErrApplicationNotFound
		}
		return nil, err
	}
	return &app, nil
}

// ResubmitApplication replaces the applicant blobs and moves the application
// back to pending — legal only from needs_revision, enforced in the WHERE
// clause.
func (r *ApplicationRepository) ResubmitApplication(ctx context.Context, id uuid.UUID, personalExtra, academic, documents json.RawMessage) (*management.ApplicationDetail, error) {
	res, err := r.db.ExecContext(ctx, `
		UPDATE applications
		   SET personal_extra = $2, academic = $3, documents = $4,
		       status = 'pending', reviewed_by = NULL, reviewed_at = NULL, review_notes = NULL
		 WHERE id = $1 AND status = 'needs_revision'`,
		id, personalExtra, academic, documents,
	)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, r.classifyApplicationMiss(ctx, id, management.ErrApplicationCannotUpdate)
	}
	return r.GetApplication(ctx, id)
}

// WithdrawApplication withdraws the application — legal only from pending or
// needs_revision, enforced in the WHERE clause.
func (r *ApplicationRepository) WithdrawApplication(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE applications
		   SET status = 'withdrawn'
		 WHERE id = $1 AND status IN ('pending', 'needs_revision')`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return r.classifyApplicationMiss(ctx, id, management.ErrApplicationCannotWithdraw)
	}
	return nil
}

// ReviewApplication decides a pending application in one transaction. The
// status precondition lives in the UPDATE's WHERE clause; an approval inserts
// the student record before the commit, so an approved applicant without a
// student record cannot exist — a duplicate student rolls the whole review
// back as ErrDuplicateStudent.
func (r *ApplicationRepository) ReviewApplication(ctx context.Context, id, reviewerID uuid.UUID, status management.ApplicationStatus, notes *string) (*management.ApplicationDetail, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var (
		userID        *uuid.UUID
		programID     uuid.UUID
		admissionYear int
		shift         management.Shift
		tuition       management.Tuition
	)
	err = tx.QueryRowxContext(ctx, `
		UPDATE applications
		   SET status = $2, reviewed_by = $3, reviewed_at = NOW(), review_notes = $4
		 WHERE id = $1 AND status = 'pending'
		RETURNING user_id, program_id, admission_year, shift, tuition`,
		id, status, reviewerID, notes,
	).Scan(&userID, &programID, &admissionYear, &shift, &tuition)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, r.classifyApplicationMiss(ctx, id, management.ErrApplicationCannotReview)
	}
	if err != nil {
		return nil, err
	}

	if status == management.ApplicationApproved && userID != nil {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO students (user_id, program_id, admission_year, current_cohort_year, current_year, shift, tuition, status)
			VALUES ($1, $2, $3, $3, 1, $4, $5, 'active')`,
			*userID, programID, admissionYear, shift, tuition,
		)
		if isUniqueViolation(err) {
			return nil, management.ErrDuplicateStudent
		}
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetApplication(ctx, id)
}

// ListApplications pages through applications matching the filter.
func (r *ApplicationRepository) ListApplications(ctx context.Context, params pagination.PageParams, filter management.ApplicationFilter) ([]management.ApplicationDetail, bool, error) {
	var conditions []string
	var args []any
	argN := 1

	if params.Cursor != "" {
		createdAt, id, err := pagination.DecodeCursor(params.Cursor)
		if err != nil {
			return nil, false, err
		}
		conditions = append(conditions, fmt.Sprintf("(a.created_at, a.id) < ($%d, $%d)", argN, argN+1))
		args = append(args, createdAt, id)
		argN += 2
	}
	if params.Query != "" {
		conditions = append(conditions, fmt.Sprintf(
			"EXISTS (SELECT 1 FROM users u2 WHERE u2.id = a.user_id AND (u2.email ILIKE $%d OR u2.full_name_en ILIKE $%d OR u2.full_name_local ILIKE $%d))",
			argN, argN, argN,
		))
		args = append(args, "%"+pagination.EscapeLike(params.Query)+"%")
		argN++
	}
	if filter.ProgramID != nil {
		conditions = append(conditions, fmt.Sprintf("a.program_id = $%d", argN))
		args = append(args, *filter.ProgramID)
		argN++
	}
	if filter.DepartmentID != nil {
		conditions = append(conditions, fmt.Sprintf("p.department_id = $%d", argN))
		args = append(args, *filter.DepartmentID)
		argN++
	}
	if filter.CollegeID != nil {
		conditions = append(conditions, fmt.Sprintf("d.college_id = $%d", argN))
		args = append(args, *filter.CollegeID)
		argN++
	}
	if filter.Scope.ProgramID != nil {
		conditions = append(conditions, fmt.Sprintf("a.program_id = $%d", argN))
		args = append(args, *filter.Scope.ProgramID)
		argN++
	}
	if filter.Scope.DepartmentID != nil {
		conditions = append(conditions, fmt.Sprintf("p.department_id = $%d", argN))
		args = append(args, *filter.Scope.DepartmentID)
		argN++
	}
	if filter.Scope.CollegeID != nil {
		conditions = append(conditions, fmt.Sprintf("d.college_id = $%d", argN))
		args = append(args, *filter.Scope.CollegeID)
		argN++
	}
	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("a.status = $%d", argN))
		args = append(args, *filter.Status)
		argN++
	}
	if filter.AdmissionYear != nil {
		conditions = append(conditions, fmt.Sprintf("a.admission_year = $%d", argN))
		args = append(args, *filter.AdmissionYear)
		argN++
	}
	if filter.Shift != nil {
		conditions = append(conditions, fmt.Sprintf("a.shift = $%d", argN))
		args = append(args, *filter.Shift)
		argN++
	}
	if filter.Tuition != nil {
		conditions = append(conditions, fmt.Sprintf("a.tuition = $%d", argN))
		args = append(args, *filter.Tuition)
		argN++
	}
	if filter.Nationality != nil {
		conditions = append(conditions, fmt.Sprintf("a.nationality = $%d", argN))
		args = append(args, *filter.Nationality)
		argN++
	}
	if filter.Gender != nil {
		conditions = append(conditions, fmt.Sprintf("a.gender = $%d", argN))
		args = append(args, *filter.Gender)
		argN++
	}
	if filter.UserID != nil {
		conditions = append(conditions, fmt.Sprintf("a.user_id = $%d", argN))
		args = append(args, *filter.UserID)
		argN++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}
	query := fmt.Sprintf("%s %s ORDER BY a.created_at DESC, a.id DESC LIMIT $%d", applicationDetailSelect, where, argN)
	args = append(args, params.Limit+1)

	var apps []management.ApplicationDetail
	if err := r.db.SelectContext(ctx, &apps, query, args...); err != nil {
		return nil, false, err
	}

	hasMore := len(apps) > params.Limit
	if hasMore {
		apps = apps[:params.Limit]
	}
	return apps, hasMore, nil
}

// ListApplicationsByUser pages through one user's applications.
func (r *ApplicationRepository) ListApplicationsByUser(ctx context.Context, userID uuid.UUID, params pagination.PageParams) ([]management.ApplicationDetail, bool, error) {
	conditions := []string{"a.user_id = $1"}
	args := []any{userID}
	argN := 2

	if params.Cursor != "" {
		createdAt, id, err := pagination.DecodeCursor(params.Cursor)
		if err != nil {
			return nil, false, err
		}
		conditions = append(conditions, fmt.Sprintf("(a.created_at, a.id) < ($%d, $%d)", argN, argN+1))
		args = append(args, createdAt, id)
		argN += 2
	}

	query := fmt.Sprintf("%s WHERE %s ORDER BY a.created_at DESC, a.id DESC LIMIT $%d",
		applicationDetailSelect, strings.Join(conditions, " AND "), argN)
	args = append(args, params.Limit+1)

	var apps []management.ApplicationDetail
	if err := r.db.SelectContext(ctx, &apps, query, args...); err != nil {
		return nil, false, err
	}

	hasMore := len(apps) > params.Limit
	if hasMore {
		apps = apps[:params.Limit]
	}
	return apps, hasMore, nil
}

// GetApplicationProgramHierarchy locates a program in the university
// structure.
func (r *ApplicationRepository) GetApplicationProgramHierarchy(ctx context.Context, programID uuid.UUID) (*management.ApplicationProgramHierarchy, error) {
	var h management.ApplicationProgramHierarchy
	err := r.db.GetContext(ctx, &h, `
		SELECT p.id as program_id, p.department_id, d.college_id
		FROM programs p
		JOIN departments d ON p.department_id = d.id
		WHERE p.id = $1`, programID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, management.ErrProgramNotFound
	}
	if err != nil {
		return nil, err
	}
	return &h, nil
}

// IsProgramActive reports whether the program accepts applications.
func (r *ApplicationRepository) IsProgramActive(ctx context.Context, programID uuid.UUID) (bool, error) {
	var isActive bool
	err := r.db.GetContext(ctx, &isActive, `SELECT is_active FROM programs WHERE id = $1`, programID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, management.ErrProgramNotFound
	}
	return isActive, err
}

// GetProgramAgeRequirements returns the program's admission age window.
func (r *ApplicationRepository) GetProgramAgeRequirements(ctx context.Context, programID uuid.UUID) (*management.ProgramAgeRequirements, error) {
	var req management.ProgramAgeRequirements
	err := r.db.QueryRowxContext(ctx, `SELECT min_age, max_age FROM programs WHERE id = $1`, programID).Scan(&req.MinAge, &req.MaxAge)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, management.ErrProgramNotFound
	}
	if err != nil {
		return nil, err
	}
	return &req, nil
}

// classifyApplicationMiss turns a guarded-update miss into the precise
// sentinel: a missing row is not-found, otherwise stateErr.
func (r *ApplicationRepository) classifyApplicationMiss(ctx context.Context, id uuid.UUID, stateErr error) error {
	var exists bool
	if err := r.db.GetContext(ctx, &exists, `SELECT EXISTS(SELECT 1 FROM applications WHERE id = $1)`, id); err != nil {
		return err
	}
	if !exists {
		return management.ErrApplicationNotFound
	}
	return stateErr
}
