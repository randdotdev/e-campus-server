package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/randdotdev/e-campus-server/internal/management"
	"github.com/randdotdev/e-campus-server/internal/shared/pagination"
)

// EnrollmentRepository is the SQL adapter for enrollments, groups, and
// enrollment requests. One adapter backs the enrollment, group, and request
// ports (plus the semester-side enrollment reads) over one connection; its
// methods are spread across enrollment.go, group.go, and
// enrollment_request.go by noun.
type EnrollmentRepository struct {
	db *sqlx.DB
}

// NewEnrollmentRepository wires the enrollment adapter.
func NewEnrollmentRepository(db *sqlx.DB) *EnrollmentRepository {
	return &EnrollmentRepository{db: db}
}

var (
	_ management.EnrollmentRepository       = (*EnrollmentRepository)(nil)
	_ management.CohortGroupRepository      = (*EnrollmentRepository)(nil)
	_ management.RequestRepository          = (*EnrollmentRepository)(nil)
	_ management.SemesterEnrollmentProvider = (*EnrollmentRepository)(nil)
	_ management.LeaveEnrollmentWithdrawer  = (*EnrollmentRepository)(nil)
)

// ── Enrollments (management.EnrollmentRepository) ─────────────────────────────

// CreateEnrollment inserts an enrollment. The unique (offering, student)
// constraint is the duplicate guard.
func (r *EnrollmentRepository) CreateEnrollment(ctx context.Context, e *management.Enrollment) error {
	err := r.db.QueryRowxContext(ctx, `
		INSERT INTO course_enrollments (offering_id, student_id, enrollment_type, status)
		VALUES ($1, $2, $3, $4)
		RETURNING id, enrolled_at`,
		e.OfferingID, e.StudentID, e.EnrollmentType, e.Status,
	).Scan(&e.ID, &e.EnrolledAt)
	if isUniqueViolation(err) {
		return management.ErrAlreadyEnrolled
	}
	if isForeignKeyViolation(err) {
		return management.ErrOfferingNotFound
	}
	return err
}

// GetEnrollment fetches the (offering, student) enrollment, or nil when none
// exists.
func (r *EnrollmentRepository) GetEnrollment(ctx context.Context, offeringID, studentID uuid.UUID) (*management.Enrollment, error) {
	var e management.Enrollment
	err := r.db.GetContext(ctx, &e,
		`SELECT * FROM course_enrollments WHERE offering_id = $1 AND student_id = $2`,
		offeringID, studentID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// ListEnrollments pages through enrollments with student display columns.
func (r *EnrollmentRepository) ListEnrollments(ctx context.Context, params pagination.PageParams, filter management.EnrollmentFilter) ([]management.EnrollmentWithStudent, bool, error) {
	var conditions []string
	var args []any
	argN := 1

	if params.Cursor != "" {
		createdAt, id, err := pagination.DecodeCursor(params.Cursor)
		if err != nil {
			return nil, false, err
		}
		conditions = append(conditions, fmt.Sprintf("(e.enrolled_at, e.id) < ($%d, $%d)", argN, argN+1))
		args = append(args, createdAt, id)
		argN += 2
	}
	if filter.Query != "" {
		conditions = append(conditions, fmt.Sprintf("(u.full_name_en ILIKE $%d OR u.full_name_local ILIKE $%d OR u.email ILIKE $%d)", argN, argN, argN))
		args = append(args, "%"+pagination.EscapeLike(filter.Query)+"%")
		argN++
	}
	if filter.OfferingID != nil {
		conditions = append(conditions, fmt.Sprintf("e.offering_id = $%d", argN))
		args = append(args, *filter.OfferingID)
		argN++
	}
	if filter.EnrollmentType != nil {
		conditions = append(conditions, fmt.Sprintf("e.enrollment_type = $%d", argN))
		args = append(args, *filter.EnrollmentType)
		argN++
	}
	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("e.status = $%d", argN))
		args = append(args, *filter.Status)
		argN++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}
	query := fmt.Sprintf(`
		SELECT
			e.id, e.offering_id, e.student_id, e.enrollment_type, e.status,
			e.enrolled_at, e.completed_at, e.final_grade,
			u.full_name_en  AS student_full_name_en,
			u.full_name_local AS student_full_name_local,
			u.email         AS student_email
		FROM course_enrollments e
		JOIN users u ON u.id = e.student_id
		%s ORDER BY e.enrolled_at DESC, e.id DESC LIMIT $%d`,
		where, argN,
	)
	args = append(args, params.Limit+1)

	var enrollments []management.EnrollmentWithStudent
	if err := r.db.SelectContext(ctx, &enrollments, query, args...); err != nil {
		return nil, false, err
	}

	hasMore := len(enrollments) > params.Limit
	if hasMore {
		enrollments = enrollments[:params.Limit]
	}
	return enrollments, hasMore, nil
}

// IsEnrolled reports whether the student is actively enrolled in the
// offering.
func (r *EnrollmentRepository) IsEnrolled(ctx context.Context, offeringID, studentID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists,
		`SELECT EXISTS(SELECT 1 FROM course_enrollments WHERE offering_id = $1 AND student_id = $2 AND status = 'enrolled')`,
		offeringID, studentID,
	)
	return exists, err
}

// GetEnrolledStudentIDs returns the actively enrolled student IDs of an
// offering.
func (r *EnrollmentRepository) GetEnrolledStudentIDs(ctx context.Context, offeringID uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := r.db.SelectContext(ctx, &ids,
		`SELECT student_id FROM course_enrollments WHERE offering_id = $1 AND status = 'enrolled'`,
		offeringID,
	)
	return ids, err
}

// GetMyEnrollments returns a student's course list, optionally filtered by
// status. Enrollment history stays visible even when its offering was
// soft-deleted — the record of study is not erased by catalogue changes.
func (r *EnrollmentRepository) GetMyEnrollments(ctx context.Context, studentID uuid.UUID, status *management.EnrollmentStatus) ([]management.MyEnrollment, error) {
	q := strings.Builder{}
	args := []any{studentID}

	q.WriteString(`
		SELECT
			e.id,
			e.offering_id,
			c.name_en AS course_name,
			c.code AS course_code,
			s.semester AS semester_name,
			e.enrollment_type,
			e.status,
			e.enrolled_at,
			e.completed_at,
			e.final_grade
		FROM course_enrollments e
		JOIN course_offerings o ON o.id = e.offering_id
		JOIN courses c ON c.id = o.course_id
		JOIN semesters s ON s.id = o.semester_id
		WHERE e.student_id = $1`)

	if status != nil {
		q.WriteString(` AND e.status = $2`)
		args = append(args, *status)
	}
	q.WriteString(` ORDER BY e.enrolled_at DESC`)

	var enrollments []management.MyEnrollment
	err := r.db.SelectContext(ctx, &enrollments, q.String(), args...)
	return enrollments, err
}

// DropEnrollment moves an enrollment to dropped — legal only from enrolled,
// enforced in the WHERE clause.
func (r *EnrollmentRepository) DropEnrollment(ctx context.Context, enrollmentID uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE course_enrollments SET status = 'dropped' WHERE id = $1 AND status = 'enrolled'`,
		enrollmentID,
	)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return management.ErrEnrollmentNotFound
	}
	return nil
}

// ── Semester-side enrollment reads (management.SemesterEnrollmentProvider) ────

// CreateStudentEnrollment enrolls a student directly during bulk enrollment.
func (r *EnrollmentRepository) CreateStudentEnrollment(ctx context.Context, offeringID, studentID uuid.UUID, enrollmentType management.EnrollmentType) error {
	e := &management.Enrollment{
		OfferingID:     offeringID,
		StudentID:      studentID,
		EnrollmentType: enrollmentType,
		Status:         management.EnrollmentEnrolled,
	}
	return r.CreateEnrollment(ctx, e)
}

// HasApprovedPretake reports whether the student has an approved pretake for
// the course in the semester.
func (r *EnrollmentRepository) HasApprovedPretake(ctx context.Context, studentID, courseID, semesterID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists, `
		SELECT EXISTS(
			SELECT 1 FROM enrollment_requests
			WHERE student_id = $1 AND course_id = $2 AND semester_id = $3
			  AND type = 'pretake' AND status = 'approved'
		)`, studentID, courseID, semesterID)
	return exists, err
}

// WasFailed reports whether the student ever failed the course.
func (r *EnrollmentRepository) WasFailed(ctx context.Context, studentID, courseID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists, `
		SELECT EXISTS(
			SELECT 1 FROM course_enrollments e
			JOIN course_offerings o ON e.offering_id = o.id
			WHERE e.student_id = $1 AND o.course_id = $2 AND e.status = 'failed'
		)`, studentID, courseID)
	return exists, err
}

// SumCredits totals the course credits of a student's enrollments in a
// semester with the given status.
func (r *EnrollmentRepository) SumCredits(ctx context.Context, studentID, semesterID uuid.UUID, status management.EnrollmentStatus) (int, error) {
	var credits int
	err := r.db.GetContext(ctx, &credits, `
		SELECT COALESCE(SUM(c.credits), 0)
		FROM course_enrollments e
		JOIN course_offerings o ON e.offering_id = o.id
		JOIN courses c ON o.course_id = c.id
		WHERE e.student_id = $1 AND o.semester_id = $2 AND e.status = $3`,
		studentID, semesterID, status,
	)
	return credits, err
}

// GetPassedCourseIDs returns the IDs of courses the student has completed.
func (r *EnrollmentRepository) GetPassedCourseIDs(ctx context.Context, studentID uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := r.db.SelectContext(ctx, &ids, `
		SELECT DISTINCT o.course_id
		FROM course_enrollments e
		JOIN course_offerings o ON e.offering_id = o.id
		WHERE e.student_id = $1 AND e.status = 'completed'`,
		studentID,
	)
	return ids, err
}

// GetRetakeRequestInfos returns the student's approved retake requests for a
// semester in the shape the semester service consumes.
func (r *EnrollmentRepository) GetRetakeRequestInfos(ctx context.Context, studentID, semesterID uuid.UUID) ([]management.AcademicRetakeRequestInfo, error) {
	var courseIDs []uuid.UUID
	err := r.db.SelectContext(ctx, &courseIDs, `
		SELECT course_id FROM enrollment_requests
		WHERE student_id = $1 AND semester_id = $2
		  AND type = 'retake' AND status = 'approved'`,
		studentID, semesterID,
	)
	if err != nil {
		return nil, err
	}
	result := make([]management.AcademicRetakeRequestInfo, len(courseIDs))
	for i, id := range courseIDs {
		result[i] = management.AcademicRetakeRequestInfo{CourseID: id}
	}
	return result, nil
}

// ── Leave-side write (management.LeaveEnrollmentWithdrawer) ───────────────────

// WithdrawEnrollmentsForLeave moves a student's active enrollments in the
// covered semesters to withdrawn_leave, one atomic statement.
func (r *EnrollmentRepository) WithdrawEnrollmentsForLeave(ctx context.Context, userID uuid.UUID, semesterIDs []uuid.UUID) error {
	if len(semesterIDs) == 0 {
		return nil
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE course_enrollments
		SET status = 'withdrawn_leave'
		WHERE student_id = $1
		  AND status = 'enrolled'
		  AND offering_id IN (
			SELECT id FROM course_offerings WHERE semester_id = ANY($2)
		  )`,
		userID, pq.Array(semesterIDs),
	)
	return err
}

// ── Reads consumed by classroom and authz (their ports; migration pending) ────

// ── Final grades (classroom's GradeWriter, adapted in main.go) ────────────────

// SetFinalGrade lands one student's computed grade on their enrollment;
// only an active enrollment takes it (Shape 2 — the status guard is the
// WHERE clause).
func (r *EnrollmentRepository) SetFinalGrade(ctx context.Context, offeringID, studentID uuid.UUID, grade float64, status string) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE course_enrollments
		SET final_grade = $1, status = $2, completed_at = NOW()
		WHERE offering_id = $3 AND student_id = $4 AND status IN ('enrolled', 'completed', 'failed')`,
		grade, status, offeringID, studentID)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return management.ErrEnrollmentNotFound
	}
	return nil
}

// ClearFinalGrades reopens the offering's grading: graded enrollments go
// back to enrolled.
func (r *EnrollmentRepository) ClearFinalGrades(ctx context.Context, offeringID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE course_enrollments
		SET final_grade = NULL, status = 'enrolled', completed_at = NULL
		WHERE offering_id = $1 AND status IN ('completed', 'failed')`, offeringID)
	return err
}

// IsOfferingFinalized: no active enrollments remain — every one carries an
// outcome.
func (r *EnrollmentRepository) IsOfferingFinalized(ctx context.Context, offeringID uuid.UUID) (bool, error) {
	var finalized bool
	err := r.db.GetContext(ctx, &finalized, `
		SELECT NOT EXISTS(
			SELECT 1 FROM course_enrollments WHERE offering_id = $1 AND status = 'enrolled')
		AND EXISTS(
			SELECT 1 FROM course_enrollments WHERE offering_id = $1)`, offeringID)
	return finalized, err
}

// GetFinalGrades is the offering's grade sheet
// (course_enrollments ⋈ users).
func (r *EnrollmentRepository) GetFinalGrades(ctx context.Context, offeringID uuid.UUID) ([]management.FinalGradeRow, error) {
	rows := []management.FinalGradeRow{}
	err := r.db.SelectContext(ctx, &rows, `
		SELECT e.student_id, u.full_name_en AS student_name, e.final_grade, e.status
		FROM course_enrollments e
		JOIN users u ON u.id = e.student_id
		WHERE e.offering_id = $1
		ORDER BY u.full_name_en`, offeringID)
	return rows, err
}

// GetEnrolledOfferingsForUser returns the offering IDs a student is enrolled
// in. It satisfies authz.CourseEnrollmentReader.
func (r *EnrollmentRepository) GetEnrolledOfferingsForUser(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := r.db.SelectContext(ctx, &ids,
		`SELECT offering_id FROM course_enrollments WHERE student_id = $1 AND status = 'enrolled'`,
		userID,
	)
	return ids, err
}
