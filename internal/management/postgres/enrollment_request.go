package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/management"
)

// ── Enrollment requests (management.RequestRepository) ────────────────────────

// CreateRequest inserts a request. The unique (student, course, semester,
// type) constraint is the duplicate guard.
func (r *EnrollmentRepository) CreateRequest(ctx context.Context, req *management.EnrollmentRequest) error {
	err := r.db.QueryRowxContext(ctx, `
		INSERT INTO enrollment_requests (type, student_id, course_id, semester_id, reason)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, status, created_at`,
		req.Type, req.StudentID, req.CourseID, req.SemesterID, req.Reason,
	).Scan(&req.ID, &req.Status, &req.CreatedAt)
	if isUniqueViolation(err) {
		return management.ErrDuplicateRequest
	}
	return err
}

// GetRequest fetches one request.
func (r *EnrollmentRepository) GetRequest(ctx context.Context, id uuid.UUID) (*management.EnrollmentRequest, error) {
	var req management.EnrollmentRequest
	if err := r.db.GetContext(ctx, &req, `SELECT * FROM enrollment_requests WHERE id = $1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, management.ErrRequestNotFound
		}
		return nil, err
	}
	return &req, nil
}

// ListRequests returns requests matching the filter, newest first.
func (r *EnrollmentRepository) ListRequests(ctx context.Context, filter management.RequestFilter) ([]management.EnrollmentRequest, error) {
	var conditions []string
	var args []any
	argN := 1

	if filter.StudentID != nil {
		conditions = append(conditions, fmt.Sprintf("student_id = $%d", argN))
		args = append(args, *filter.StudentID)
		argN++
	}
	if filter.CourseID != nil {
		conditions = append(conditions, fmt.Sprintf("course_id = $%d", argN))
		args = append(args, *filter.CourseID)
		argN++
	}
	if filter.SemesterID != nil {
		conditions = append(conditions, fmt.Sprintf("semester_id = $%d", argN))
		args = append(args, *filter.SemesterID)
		argN++
	}
	if filter.Type != nil {
		conditions = append(conditions, fmt.Sprintf("type = $%d", argN))
		args = append(args, *filter.Type)
		argN++
	}
	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argN))
		args = append(args, *filter.Status)
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}
	query := fmt.Sprintf("SELECT * FROM enrollment_requests %s ORDER BY created_at DESC", where)

	var requests []management.EnrollmentRequest
	if err := r.db.SelectContext(ctx, &requests, query, args...); err != nil {
		return nil, err
	}
	return requests, nil
}

// ApproveRequest approves a pending request — the precondition is the WHERE
// clause, one atomic statement.
func (r *EnrollmentRepository) ApproveRequest(ctx context.Context, id, reviewerID uuid.UUID) error {
	return r.decideRequest(ctx, `
		UPDATE enrollment_requests
		SET status = 'approved', reviewed_by = $2, reviewed_at = NOW()
		WHERE id = $1 AND status = 'pending'`, id, reviewerID)
}

// RejectRequest rejects a pending request with a reason — the precondition is
// the WHERE clause, one atomic statement.
func (r *EnrollmentRepository) RejectRequest(ctx context.Context, id, reviewerID uuid.UUID, reason string) error {
	return r.decideRequest(ctx, `
		UPDATE enrollment_requests
		SET status = 'rejected', reviewed_by = $2, reviewed_at = NOW(), rejection_reason = $3
		WHERE id = $1 AND status = 'pending'`, id, reviewerID, reason)
}

func (r *EnrollmentRepository) decideRequest(ctx context.Context, query string, id uuid.UUID, args ...any) error {
	result, err := r.db.ExecContext(ctx, query, append([]any{id}, args...)...)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		var exists bool
		if err := r.db.GetContext(ctx, &exists, `SELECT EXISTS(SELECT 1 FROM enrollment_requests WHERE id = $1)`, id); err != nil {
			return err
		}
		if !exists {
			return management.ErrRequestNotFound
		}
		return management.ErrAlreadyReviewed
	}
	return nil
}

// ── Request-side reads ────────────────────────────────────────────────────────

// CourseExists reports whether the course exists (soft-deleted courses do
// not).
func (r *EnrollmentRepository) CourseExists(ctx context.Context, id uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists,
		`SELECT EXISTS(SELECT 1 FROM courses WHERE id = $1 AND deleted_at IS NULL)`, id)
	return exists, err
}

// SemesterExists reports whether the semester exists (soft-deleted semesters
// do not).
func (r *EnrollmentRepository) SemesterExists(ctx context.Context, id uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists,
		`SELECT EXISTS(SELECT 1 FROM semesters WHERE id = $1 AND deleted_at IS NULL)`, id)
	return exists, err
}

// IsSemesterActive reports whether the semester is currently active; a
// missing semester is simply not active.
func (r *EnrollmentRepository) IsSemesterActive(ctx context.Context, semesterID uuid.UUID) (bool, error) {
	var status management.SemesterStatus
	err := r.db.GetContext(ctx, &status,
		`SELECT status FROM semesters WHERE id = $1 AND deleted_at IS NULL`, semesterID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return status == management.SemesterActive, nil
}

// GetCoursePrerequisite returns the course's prerequisite ID, or nil when the
// course has none.
func (r *EnrollmentRepository) GetCoursePrerequisite(ctx context.Context, courseID uuid.UUID) (*uuid.UUID, error) {
	var prereqID *uuid.UUID
	err := r.db.GetContext(ctx, &prereqID, `SELECT requires FROM courses WHERE id = $1`, courseID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, management.ErrCourseNotFound
	}
	return prereqID, err
}

// GetPrereqStatus derives the student's standing on a prerequisite course
// from the latest enrollment attempt.
func (r *EnrollmentRepository) GetPrereqStatus(ctx context.Context, studentID, prereqCourseID uuid.UUID) (*management.PrereqStatus, error) {
	var status management.PrereqStatus
	err := r.db.QueryRowxContext(ctx,
		`SELECT id, code, name_en, name_local FROM courses WHERE id = $1`,
		prereqCourseID,
	).Scan(&status.CourseID, &status.CourseCode, &status.CourseNameEN, &status.CourseNameLocal)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, management.ErrCourseNotFound
	}
	if err != nil {
		return nil, err
	}

	takeStatus, err := r.latestTakeStatus(ctx, studentID, prereqCourseID)
	if err != nil {
		return nil, err
	}
	status.Status = takeStatus
	return &status, nil
}

// GetCourseTakeStatus derives the student's standing on a course plus the
// natural-cohort check.
func (r *EnrollmentRepository) GetCourseTakeStatus(ctx context.Context, studentID, courseID uuid.UUID) (*management.CourseTakeStatus, error) {
	var status management.CourseTakeStatus
	err := r.db.QueryRowxContext(ctx,
		`SELECT id, code, name_en, name_local FROM courses WHERE id = $1`,
		courseID,
	).Scan(&status.CourseID, &status.CourseCode, &status.CourseNameEN, &status.CourseNameLocal)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, management.ErrCourseNotFound
	}
	if err != nil {
		return nil, err
	}

	takeStatus, err := r.latestTakeStatus(ctx, studentID, courseID)
	if err != nil {
		return nil, err
	}
	status.Status = takeStatus

	// Natural cohort: the course is offered to the student's current cohort.
	err = r.db.GetContext(ctx, &status.IsNaturalCohort, `
		SELECT EXISTS(
			SELECT 1 FROM students s
			JOIN course_offerings o ON o.cohort_year = s.current_cohort_year
			WHERE s.user_id = $1 AND o.course_id = $2
		)`, studentID, courseID)
	if err != nil {
		return nil, err
	}
	return &status, nil
}

// latestTakeStatus maps the student's most recent enrollment attempt at a
// course onto a TakeStatus.
func (r *EnrollmentRepository) latestTakeStatus(ctx context.Context, studentID, courseID uuid.UUID) (management.TakeStatus, error) {
	var enrollStatus management.EnrollmentStatus
	err := r.db.GetContext(ctx, &enrollStatus, `
		SELECT e.status FROM course_enrollments e
		JOIN course_offerings o ON e.offering_id = o.id
		WHERE e.student_id = $1 AND o.course_id = $2
		ORDER BY e.enrolled_at DESC
		LIMIT 1`, studentID, courseID)
	if errors.Is(err, sql.ErrNoRows) {
		return management.TakeNotTaken, nil
	}
	if err != nil {
		return management.TakeNotTaken, err
	}

	switch enrollStatus {
	case management.EnrollmentCompleted:
		return management.TakePassed, nil
	case management.EnrollmentEnrolled:
		return management.TakeInProgress, nil
	case management.EnrollmentFailed:
		return management.TakeFailed, nil
	default:
		return management.TakeNotTaken, nil
	}
}

// GetStudentCohortInfo returns the cohort year and shift of the student
// identified by user ID.
func (r *EnrollmentRepository) GetStudentCohortInfo(ctx context.Context, studentID uuid.UUID) (cohortYear int, shift management.Shift, err error) {
	err = r.db.QueryRowxContext(ctx,
		`SELECT current_cohort_year, shift FROM students WHERE user_id = $1`,
		studentID,
	).Scan(&cohortYear, &shift)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, "", management.ErrStudentNotFound
	}
	return cohortYear, shift, err
}

// GetOfferingIDForEnrollment returns the live offering matching the request's
// course, semester, cohort, and shift, or nil when none exists.
func (r *EnrollmentRepository) GetOfferingIDForEnrollment(ctx context.Context, courseID, semesterID uuid.UUID, cohortYear int, shift management.Shift) (*uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.GetContext(ctx, &id, `
		SELECT id FROM course_offerings
		WHERE course_id = $1 AND semester_id = $2 AND cohort_year = $3 AND shift = $4 AND deleted_at IS NULL`,
		courseID, semesterID, cohortYear, shift,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// GetStudentName returns the display name of the user behind a request.
func (r *EnrollmentRepository) GetStudentName(ctx context.Context, studentID uuid.UUID) (string, error) {
	var name string
	err := r.db.GetContext(ctx, &name, `SELECT full_name_en FROM users WHERE id = $1`, studentID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", management.ErrUserNotFound
	}
	return name, err
}
