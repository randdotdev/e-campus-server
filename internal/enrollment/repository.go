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
	"github.com/ranjdotdev/e-campus-server/internal/pagination"
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505"
}

// Enrollment operations

func (r *Repository) CreateEnrollment(ctx context.Context, e *Enrollment) error {
	query := `
		INSERT INTO course_enrollments (offering_id, student_id, enrollment_type, status)
		VALUES ($1, $2, $3, $4)
		RETURNING id, enrolled_at`

	enrollmentType := e.EnrollmentType
	if enrollmentType == "" {
		enrollmentType = EnrollmentTypeCurriculum
	}

	status := e.Status
	if status == "" {
		status = EnrollmentStatusEnrolled
	}

	return r.db.QueryRowxContext(ctx, query,
		e.OfferingID, e.StudentID, enrollmentType, status,
	).Scan(&e.ID, &e.EnrolledAt)
}

func (r *Repository) GetEnrollment(ctx context.Context, offeringID, studentID uuid.UUID) (*Enrollment, error) {
	var enrollment Enrollment
	query := `SELECT * FROM course_enrollments WHERE offering_id = $1 AND student_id = $2`

	if err := r.db.GetContext(ctx, &enrollment, query, offeringID, studentID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &enrollment, nil
}

func (r *Repository) ListEnrollments(ctx context.Context, params pagination.PageParams, filters EnrollmentFilters) ([]Enrollment, bool, error) {
	query := strings.Builder{}
	args := []any{}
	argN := 1

	query.WriteString("SELECT e.* FROM course_enrollments e")

	if filters.Query != "" {
		query.WriteString(" JOIN users u ON e.student_id = u.id")
	}

	query.WriteString(" WHERE 1=1")

	if params.Cursor != "" {
		createdAt, id, err := pagination.DecodeCursor(params.Cursor)
		if err != nil {
			return nil, false, err
		}
		query.WriteString(fmt.Sprintf(" AND (e.enrolled_at, e.id) < ($%d, $%d)", argN, argN+1))
		args = append(args, createdAt, id)
		argN += 2
	}

	if filters.Query != "" {
		query.WriteString(fmt.Sprintf(" AND (u.full_name_en ILIKE $%d OR u.full_name_local ILIKE $%d OR u.email ILIKE $%d)", argN, argN, argN))
		args = append(args, "%"+pagination.EscapeLike(filters.Query)+"%")
		argN++
	}

	if filters.OfferingID != nil {
		query.WriteString(fmt.Sprintf(" AND e.offering_id = $%d", argN))
		args = append(args, *filters.OfferingID)
		argN++
	}

	if filters.EnrollmentType != nil {
		query.WriteString(fmt.Sprintf(" AND e.enrollment_type = $%d", argN))
		args = append(args, *filters.EnrollmentType)
		argN++
	}

	if filters.Status != nil {
		query.WriteString(fmt.Sprintf(" AND e.status = $%d", argN))
		args = append(args, *filters.Status)
		argN++
	}

	query.WriteString(" ORDER BY e.enrolled_at DESC, e.id DESC")
	query.WriteString(fmt.Sprintf(" LIMIT $%d", argN))
	args = append(args, params.Limit+1)

	var enrollments []Enrollment
	if err := r.db.SelectContext(ctx, &enrollments, query.String(), args...); err != nil {
		return nil, false, err
	}

	hasMore := len(enrollments) > params.Limit
	if hasMore {
		enrollments = enrollments[:params.Limit]
	}

	return enrollments, hasMore, nil
}

func (r *Repository) UpdateEnrollment(ctx context.Context, e *Enrollment) error {
	query := `
		UPDATE course_enrollments
		SET status = $2, completed_at = $3, final_grade = $4
		WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, e.ID, e.Status, e.CompletedAt, e.FinalGrade)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrEnrollmentNotFound
	}
	return nil
}

func (r *Repository) IsEnrolled(ctx context.Context, offeringID, studentID uuid.UUID) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM course_enrollments WHERE offering_id = $1 AND student_id = $2 AND status = 'enrolled')`
	err := r.db.GetContext(ctx, &exists, query, offeringID, studentID)
	return exists, err
}

func (r *Repository) GetEnrolledStudentIDs(ctx context.Context, offeringID uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	query := `SELECT student_id FROM course_enrollments WHERE offering_id = $1 AND status = 'enrolled'`
	err := r.db.SelectContext(ctx, &ids, query, offeringID)
	return ids, err
}

func (r *Repository) GetStudentEnrollments(ctx context.Context, studentID uuid.UUID) ([]Enrollment, error) {
	var enrollments []Enrollment
	query := `SELECT * FROM course_enrollments WHERE student_id = $1 ORDER BY enrolled_at DESC`

	if err := r.db.SelectContext(ctx, &enrollments, query, studentID); err != nil {
		return nil, err
	}
	return enrollments, nil
}

func (r *Repository) DropEnrollment(ctx context.Context, enrollmentID uuid.UUID) error {
	query := `UPDATE course_enrollments SET status = 'dropped' WHERE id = $1 AND status = 'enrolled'`
	result, err := r.db.ExecContext(ctx, query, enrollmentID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrEnrollmentNotFound
	}
	return nil
}

// Project group operations

func (r *Repository) CreateProjectGroup(ctx context.Context, g *ProjectGroup) error {
	query := `INSERT INTO project_groups (offering_id, type, name) VALUES ($1, $2, $3) RETURNING id, created_at`
	return r.db.QueryRowxContext(ctx, query, g.OfferingID, g.Type, g.Name).Scan(&g.ID, &g.CreatedAt)
}

func (r *Repository) GetProjectGroupByID(ctx context.Context, id uuid.UUID) (*ProjectGroup, error) {
	var g ProjectGroup
	err := r.db.GetContext(ctx, &g, `SELECT id, offering_id, type, name, created_at FROM project_groups WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &g, err
}

func (r *Repository) ListProjectGroups(ctx context.Context, offeringID uuid.UUID) ([]ProjectGroup, error) {
	var groups []ProjectGroup
	query := `SELECT id, offering_id, type, name, created_at FROM project_groups WHERE offering_id = $1 ORDER BY type, name`
	err := r.db.SelectContext(ctx, &groups, query, offeringID)
	return groups, err
}

func (r *Repository) DeleteProjectGroup(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM project_groups WHERE id = $1`, id)
	return err
}

func (r *Repository) ProjectGroupExists(ctx context.Context, id uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists, `SELECT EXISTS(SELECT 1 FROM project_groups WHERE id = $1)`, id)
	return exists, err
}

func (r *Repository) AssignToProjectGroup(ctx context.Context, m *ProjectGroupMember) error {
	query := `INSERT INTO project_group_members (student_id, group_id) VALUES ($1, $2) RETURNING id, assigned_at`
	return r.db.QueryRowxContext(ctx, query, m.StudentID, m.ProjectGroupID).Scan(&m.ID, &m.AssignedAt)
}

func (r *Repository) RemoveFromProjectGroup(ctx context.Context, studentID, groupID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM project_group_members WHERE student_id = $1 AND group_id = $2`, studentID, groupID)
	return err
}

func (r *Repository) GetStudentProjectGroupIDs(ctx context.Context, studentID, offeringID uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	query := `SELECT pgm.group_id FROM project_group_members pgm
		JOIN project_groups pg ON pg.id = pgm.group_id
		WHERE pgm.student_id = $1 AND pg.offering_id = $2`
	err := r.db.SelectContext(ctx, &ids, query, studentID, offeringID)
	return ids, err
}

// Cohort group operations

func (r *Repository) CreateCohortGroup(ctx context.Context, g *CohortGroup) error {
	query := `INSERT INTO cohort_groups (program_id, cohort_year, stage, type, name) VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at`
	return r.db.QueryRowxContext(ctx, query, g.ProgramID, g.CohortYear, g.Stage, g.Type, g.Name).Scan(&g.ID, &g.CreatedAt)
}

func (r *Repository) GetCohortGroupByID(ctx context.Context, id uuid.UUID) (*CohortGroup, error) {
	var g CohortGroup
	err := r.db.GetContext(ctx, &g, `SELECT * FROM cohort_groups WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &g, err
}

func (r *Repository) ListCohortGroups(ctx context.Context, programID uuid.UUID, cohortYear, stage int) ([]CohortGroup, error) {
	var groups []CohortGroup
	query := `SELECT * FROM cohort_groups WHERE program_id = $1 AND cohort_year = $2 AND stage = $3 ORDER BY type, name`
	err := r.db.SelectContext(ctx, &groups, query, programID, cohortYear, stage)
	return groups, err
}

func (r *Repository) DeleteCohortGroup(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM cohort_groups WHERE id = $1`, id)
	return err
}

func (r *Repository) CohortGroupExists(ctx context.Context, id uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists, `SELECT EXISTS(SELECT 1 FROM cohort_groups WHERE id = $1)`, id)
	return exists, err
}

func (r *Repository) AssignToCohortGroup(ctx context.Context, m *StudentCohortGroup) error {
	query := `INSERT INTO student_cohort_groups (student_id, cohort_group_id) VALUES ($1, $2) RETURNING id, assigned_at`
	return r.db.QueryRowxContext(ctx, query, m.StudentID, m.CohortGroupID).Scan(&m.ID, &m.AssignedAt)
}

func (r *Repository) RemoveFromCohortGroup(ctx context.Context, studentID, groupID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM student_cohort_groups WHERE student_id = $1 AND cohort_group_id = $2`, studentID, groupID)
	return err
}

func (r *Repository) GetStudentCohortGroupIDs(ctx context.Context, studentID uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	query := `SELECT cohort_group_id FROM student_cohort_groups WHERE student_id = $1`
	err := r.db.SelectContext(ctx, &ids, query, studentID)
	return ids, err
}

// Request operations (pretake/retake)

func (r *Repository) CreateRequest(ctx context.Context, req *Request) error {
	query := `
		INSERT INTO enrollment_requests (type, student_id, course_id, semester_id, reason)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, status, created_at`

	err := r.db.QueryRowxContext(ctx, query,
		req.Type, req.StudentID, req.CourseID, req.SemesterID, req.Reason,
	).Scan(&req.ID, &req.Status, &req.CreatedAt)

	if err != nil {
		if isUniqueViolation(err) {
			return ErrDuplicateRequest
		}
		return err
	}
	return nil
}

func (r *Repository) GetRequestByID(ctx context.Context, id uuid.UUID) (*Request, error) {
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

func (r *Repository) ListRequests(ctx context.Context, filters RequestFilters) ([]Request, error) {
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

func (r *Repository) ApproveRequest(ctx context.Context, id, reviewerID uuid.UUID) error {
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

func (r *Repository) RejectRequest(ctx context.Context, id, reviewerID uuid.UUID, reason string) error {
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

func (r *Repository) HasApprovedRequest(ctx context.Context, studentID, courseID, semesterID uuid.UUID, reqType string) (bool, error) {
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

// Lookup operations

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

func (r *Repository) GetPassedCourseIDs(ctx context.Context, studentID uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	query := `
		SELECT DISTINCT o.course_id
		FROM course_enrollments e
		JOIN course_offerings o ON e.offering_id = o.id
		WHERE e.student_id = $1 AND e.status = 'completed'`
	err := r.db.SelectContext(ctx, &ids, query, studentID)
	return ids, err
}

func (r *Repository) WasFailed(ctx context.Context, studentID, courseID uuid.UUID) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS(
			SELECT 1 FROM course_enrollments e
			JOIN course_offerings o ON e.offering_id = o.id
			WHERE e.student_id = $1 AND o.course_id = $2 AND e.status = 'failed'
		)`
	err := r.db.GetContext(ctx, &exists, query, studentID, courseID)
	return exists, err
}

func (r *Repository) SumCredits(ctx context.Context, studentID, semesterID uuid.UUID, status string) (int, error) {
	var credits int
	query := `
		SELECT COALESCE(SUM(c.credits), 0)
		FROM course_enrollments e
		JOIN course_offerings o ON e.offering_id = o.id
		JOIN courses c ON o.course_id = c.id
		WHERE e.student_id = $1 AND o.semester_id = $2 AND e.status = $3`
	err := r.db.GetContext(ctx, &credits, query, studentID, semesterID, status)
	return credits, err
}

func (r *Repository) WithdrawEnrollmentsForLeave(ctx context.Context, studentID uuid.UUID, semesterIDs []uuid.UUID) error {
	if len(semesterIDs) == 0 {
		return nil
	}
	query := `
		UPDATE course_enrollments
		SET status = 'withdrawn_leave'
		WHERE student_id = $1
		  AND status = 'enrolled'
		  AND offering_id IN (
			SELECT id FROM course_offerings WHERE semester_id = ANY($2)
		  )`
	_, err := r.db.ExecContext(ctx, query, studentID, pq.Array(semesterIDs))
	return err
}

func (r *Repository) GetApprovedRetakeRequests(ctx context.Context, studentID, semesterID uuid.UUID) ([]uuid.UUID, error) {
	var courseIDs []uuid.UUID
	query := `
		SELECT course_id FROM enrollment_requests
		WHERE student_id = $1 AND semester_id = $2
		  AND type = 'retake' AND status = 'approved'`
	err := r.db.SelectContext(ctx, &courseIDs, query, studentID, semesterID)
	return courseIDs, err
}

func (r *Repository) GetStudentCohortInfo(ctx context.Context, studentID uuid.UUID) (cohortYear int, shift string, err error) {
	query := `SELECT current_cohort_year, shift FROM students WHERE user_id = $1`
	err = r.db.QueryRowxContext(ctx, query, studentID).Scan(&cohortYear, &shift)
	return cohortYear, shift, err
}

func (r *Repository) GetOfferingIDForEnrollment(ctx context.Context, courseID, semesterID uuid.UUID, cohortYear int, shift string) (*uuid.UUID, error) {
	var id uuid.UUID
	query := `
		SELECT id FROM course_offerings
		WHERE course_id = $1 AND semester_id = $2 AND cohort_year = $3 AND shift = $4`
	err := r.db.GetContext(ctx, &id, query, courseID, semesterID, cohortYear, shift)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func (r *Repository) IsSemesterActive(ctx context.Context, semesterID uuid.UUID) (bool, error) {
	var status string
	query := `SELECT status FROM semesters WHERE id = $1`
	err := r.db.GetContext(ctx, &status, query, semesterID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return status == "active", nil
}
