package enrollment

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, req *Request) error {
	query := `
		INSERT INTO enrollment_requests (type, student_id, course_id, semester_id, reason)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, status, created_at`

	err := r.db.QueryRowxContext(ctx, query,
		req.Type, req.StudentID, req.CourseID, req.SemesterID, req.Reason,
	).Scan(&req.ID, &req.Status, &req.CreatedAt)

	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return ErrDuplicateRequest
		}
		return err
	}
	return nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Request, error) {
	var req Request
	query := `SELECT * FROM enrollment_requests WHERE id = $1`

	if err := r.db.GetContext(ctx, &req, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRequestNotFound
		}
		return nil, err
	}
	return &req, nil
}

func (r *Repository) List(ctx context.Context, filters Filters) ([]Request, error) {
	query := strings.Builder{}
	args := []any{}
	argN := 1

	query.WriteString("SELECT * FROM enrollment_requests WHERE 1=1")

	if filters.StudentID != nil {
		query.WriteString(fmt.Sprintf(" AND student_id = $%d", argN))
		args = append(args, *filters.StudentID)
		argN++
	}

	if filters.CourseID != nil {
		query.WriteString(fmt.Sprintf(" AND course_id = $%d", argN))
		args = append(args, *filters.CourseID)
		argN++
	}

	if filters.SemesterID != nil {
		query.WriteString(fmt.Sprintf(" AND semester_id = $%d", argN))
		args = append(args, *filters.SemesterID)
		argN++
	}

	if filters.Type != nil {
		query.WriteString(fmt.Sprintf(" AND type = $%d", argN))
		args = append(args, *filters.Type)
		argN++
	}

	if filters.Status != nil {
		query.WriteString(fmt.Sprintf(" AND status = $%d", argN))
		args = append(args, *filters.Status)
	}

	query.WriteString(" ORDER BY created_at DESC")

	var requests []Request
	if err := r.db.SelectContext(ctx, &requests, query.String(), args...); err != nil {
		return nil, err
	}
	return requests, nil
}

func (r *Repository) Approve(ctx context.Context, id, reviewerID uuid.UUID) error {
	query := `
		UPDATE enrollment_requests
		SET status = 'approved', reviewed_by = $2, reviewed_at = NOW()
		WHERE id = $1 AND status = 'pending'`

	result, err := r.db.ExecContext(ctx, query, id, reviewerID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrAlreadyReviewed
	}
	return nil
}

func (r *Repository) Reject(ctx context.Context, id, reviewerID uuid.UUID, reason string) error {
	query := `
		UPDATE enrollment_requests
		SET status = 'rejected', reviewed_by = $2, reviewed_at = NOW(), rejection_reason = $3
		WHERE id = $1 AND status = 'pending'`

	result, err := r.db.ExecContext(ctx, query, id, reviewerID, reason)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrAlreadyReviewed
	}
	return nil
}

func (r *Repository) HasApproved(ctx context.Context, studentID, courseID, semesterID uuid.UUID, reqType string) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS(
			SELECT 1 FROM enrollment_requests
			WHERE student_id = $1 AND course_id = $2 AND semester_id = $3
			  AND type = $4 AND status = 'approved'
		)`
	err := r.db.GetContext(ctx, &exists, query, studentID, courseID, semesterID, reqType)
	return exists, err
}

func (r *Repository) GetPrereqStatus(ctx context.Context, studentID, prereqCourseID uuid.UUID) (*PrereqStatus, error) {
	var status PrereqStatus

	courseQuery := `SELECT id, code, name_en, name_local FROM courses WHERE id = $1`
	var nameLocal *string
	err := r.db.QueryRowxContext(ctx, courseQuery, prereqCourseID).Scan(
		&status.CourseID, &status.CourseCode, &status.CourseNameEN, &nameLocal,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCourseNotFound
		}
		return nil, err
	}
	status.CourseNameLocal = nameLocal

	enrollQuery := `
		SELECT e.status FROM course_enrollments e
		JOIN course_offerings o ON e.offering_id = o.id
		WHERE e.student_id = $1 AND o.course_id = $2
		ORDER BY e.enrolled_at DESC
		LIMIT 1`

	var enrollStatus string
	err = r.db.GetContext(ctx, &enrollStatus, enrollQuery, studentID, prereqCourseID)
	if errors.Is(err, sql.ErrNoRows) {
		status.Status = PrereqNotTaken
		return &status, nil
	}
	if err != nil {
		return nil, err
	}

	switch enrollStatus {
	case "completed":
		status.Status = PrereqPassed
	case "enrolled":
		status.Status = PrereqInProgress
	case "failed":
		status.Status = PrereqFailed
	default:
		status.Status = PrereqNotTaken
	}

	return &status, nil
}

func (r *Repository) GetCourseStatus(ctx context.Context, studentID, courseID uuid.UUID) (*CourseStatus, error) {
	var status CourseStatus

	courseQuery := `SELECT id, code, name_en, name_local FROM courses WHERE id = $1`
	var nameLocal *string
	err := r.db.QueryRowxContext(ctx, courseQuery, courseID).Scan(
		&status.CourseID, &status.CourseCode, &status.CourseNameEN, &nameLocal,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCourseNotFound
		}
		return nil, err
	}
	status.CourseNameLocal = nameLocal

	enrollQuery := `
		SELECT e.status FROM course_enrollments e
		JOIN course_offerings o ON e.offering_id = o.id
		WHERE e.student_id = $1 AND o.course_id = $2
		ORDER BY e.enrolled_at DESC
		LIMIT 1`

	var enrollStatus string
	err = r.db.GetContext(ctx, &enrollStatus, enrollQuery, studentID, courseID)
	if errors.Is(err, sql.ErrNoRows) {
		status.Status = CourseNotTaken
		return &status, nil
	}
	if err != nil {
		return nil, err
	}

	switch enrollStatus {
	case "completed":
		status.Status = CoursePassed
	case "enrolled":
		status.Status = CourseInProgress
	case "failed":
		status.Status = CourseFailed
	default:
		status.Status = CourseNotTaken
	}

	status.IsNaturalCohort, err = r.IsNaturalCohort(ctx, studentID, courseID)
	if err != nil {
		return nil, err
	}

	return &status, nil
}

func (r *Repository) GetCoursePrerequisite(ctx context.Context, courseID uuid.UUID) (*uuid.UUID, error) {
	var prereqID *uuid.UUID
	query := `SELECT requires FROM courses WHERE id = $1`
	err := r.db.GetContext(ctx, &prereqID, query, courseID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCourseNotFound
	}
	return prereqID, err
}

func (r *Repository) GetStudentName(ctx context.Context, studentID uuid.UUID) (string, error) {
	var name string
	query := `SELECT full_name_en FROM users WHERE id = $1`
	err := r.db.GetContext(ctx, &name, query, studentID)
	return name, err
}

func (r *Repository) CourseExists(ctx context.Context, id uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists, `SELECT EXISTS(SELECT 1 FROM courses WHERE id = $1)`, id)
	return exists, err
}

func (r *Repository) SemesterExists(ctx context.Context, id uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists, `SELECT EXISTS(SELECT 1 FROM semesters WHERE id = $1)`, id)
	return exists, err
}

func (r *Repository) IsNaturalCohort(ctx context.Context, studentID, courseID uuid.UUID) (bool, error) {
	// Check if the student's cohort year matches the expected year for this course
	// This requires checking program_curriculum to see when this course should be taken
	// For now, simplified: check if student is in same cohort as the course offering
	query := `
		SELECT EXISTS(
			SELECT 1 FROM students s
			JOIN course_offerings o ON o.cohort_year = s.cohort_year
			JOIN courses c ON c.id = o.course_id
			WHERE s.user_id = $1 AND c.id = $2
		)`
	var exists bool
	err := r.db.GetContext(ctx, &exists, query, studentID, courseID)
	return exists, err
}
