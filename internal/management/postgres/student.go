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

const studentWithUserSelect = `
	SELECT s.user_id, s.program_id, s.admission_year,
		s.current_cohort_year, s.current_year, s.shift, s.tuition, s.status,
		s.enrolled_at, s.created_at, s.version,
		u.full_name_en AS name_en, u.full_name_local AS name_local
	FROM students s
	JOIN users u ON s.user_id = u.id`

// StudentRepository is the SQL adapter for students, leaves, and cohort
// history. One adapter backs the student, leave, and semester-side student
// ports over one connection.
type StudentRepository struct {
	db *sqlx.DB
}

// NewStudentRepository wires the student adapter.
func NewStudentRepository(db *sqlx.DB) *StudentRepository {
	return &StudentRepository{db: db}
}

var (
	_ management.StudentRepository       = (*StudentRepository)(nil)
	_ management.LeaveRepository         = (*StudentRepository)(nil)
	_ management.LeaveStudentUpdater     = (*StudentRepository)(nil)
	_ management.SemesterStudentProvider = (*StudentRepository)(nil)
)

// ── Students ──────────────────────────────────────────────────────────────────

// CreateStudent inserts a student. The user_id primary key is the duplicate
// guard; a broken program reference surfaces as ErrProgramNotFound.
func (r *StudentRepository) CreateStudent(ctx context.Context, s *management.Student) error {
	err := r.db.QueryRowxContext(ctx, `
		INSERT INTO students (user_id, program_id, admission_year, current_cohort_year, current_year, shift, tuition, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING enrolled_at, created_at, version`,
		s.UserID, s.ProgramID, s.AdmissionYear, s.CurrentCohortYear, s.CurrentYear, s.Shift, s.Tuition, s.Status,
	).Scan(&s.EnrolledAt, &s.CreatedAt, &s.Version)
	if isUniqueViolation(err) {
		return management.ErrDuplicateStudent
	}
	if isForeignKeyViolation(err) {
		return management.ErrProgramNotFound
	}
	return err
}

// GetStudent fetches one student with the user's display columns.
func (r *StudentRepository) GetStudent(ctx context.Context, userID uuid.UUID) (*management.StudentSummary, error) {
	var s management.StudentSummary
	err := r.db.GetContext(ctx, &s, studentWithUserSelect+` WHERE s.user_id = $1`, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, management.ErrStudentNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// ListStudents pages through students matching the filter, newest first.
func (r *StudentRepository) ListStudents(ctx context.Context, params pagination.PageParams, filter management.StudentFilter) ([]management.StudentSummary, bool, error) {
	joins := "JOIN users u ON s.user_id = u.id"
	if filter.CohortGroupID != nil {
		joins += " JOIN student_cohort_groups scg ON scg.student_id = s.user_id"
	}
	// The viewer's scope reaches students through the program's lineage.
	if filter.Scope.DepartmentID != nil || filter.Scope.CollegeID != nil {
		joins += " JOIN programs p ON p.id = s.program_id"
	}
	if filter.Scope.CollegeID != nil {
		joins += " JOIN departments d ON d.id = p.department_id"
	}

	var conditions []string
	var args []any
	argN := 1

	if params.Cursor != "" {
		createdAt, id, err := pagination.DecodeCursor(params.Cursor)
		if err != nil {
			return nil, false, err
		}
		conditions = append(conditions, fmt.Sprintf("(s.created_at, s.user_id) < ($%d, $%d)", argN, argN+1))
		args = append(args, createdAt, id)
		argN += 2
	}
	if filter.ProgramID != nil {
		conditions = append(conditions, fmt.Sprintf("s.program_id = $%d", argN))
		args = append(args, *filter.ProgramID)
		argN++
	}
	if filter.CohortYear != nil {
		conditions = append(conditions, fmt.Sprintf("s.current_cohort_year = $%d", argN))
		args = append(args, *filter.CohortYear)
		argN++
	}
	if filter.Stage != nil {
		conditions = append(conditions, fmt.Sprintf("s.current_year = $%d", argN))
		args = append(args, *filter.Stage)
		argN++
	}
	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("s.status = $%d", argN))
		args = append(args, *filter.Status)
		argN++
	}
	if filter.Shift != nil {
		conditions = append(conditions, fmt.Sprintf("s.shift = $%d", argN))
		args = append(args, *filter.Shift)
		argN++
	}
	if filter.CohortGroupID != nil {
		conditions = append(conditions, fmt.Sprintf("scg.cohort_group_id = $%d", argN))
		args = append(args, *filter.CohortGroupID)
		argN++
	}
	if filter.Scope.ProgramID != nil {
		conditions = append(conditions, fmt.Sprintf("s.program_id = $%d", argN))
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
	if filter.Query != nil && *filter.Query != "" {
		conditions = append(conditions, fmt.Sprintf("(u.full_name_en ILIKE $%d OR u.email ILIKE $%d)", argN, argN))
		args = append(args, "%"+pagination.EscapeLike(*filter.Query)+"%")
		argN++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}
	query := fmt.Sprintf(`
		SELECT s.user_id, s.program_id, s.admission_year,
			s.current_cohort_year, s.current_year, s.shift, s.tuition, s.status,
			s.enrolled_at, s.created_at, s.version,
			u.full_name_en AS name_en, u.full_name_local AS name_local
		FROM students s
		%s
		%s
		ORDER BY s.created_at DESC, s.user_id DESC
		LIMIT $%d`, joins, where, argN)
	args = append(args, params.Limit+1)

	var students []management.StudentSummary
	if err := r.db.SelectContext(ctx, &students, query, args...); err != nil {
		return nil, false, err
	}

	hasMore := len(students) > params.Limit
	if hasMore {
		students = students[:params.Limit]
	}
	return students, hasMore, nil
}

// UpdateStudent is an optimistic compare-and-swap keyed on version.
func (r *StudentRepository) UpdateStudent(ctx context.Context, s *management.Student, expectedVersion int64) (int64, error) {
	const query = `
		UPDATE students
		   SET current_cohort_year = $2, current_year = $3, shift = $4,
		       tuition = $5, status = $6, version = version + 1
		 WHERE user_id = $1 AND version = $7
		RETURNING version`
	var newVersion int64
	err := r.db.QueryRowxContext(ctx, query,
		s.UserID, s.CurrentCohortYear, s.CurrentYear, s.Shift, s.Tuition, s.Status, expectedVersion,
	).Scan(&newVersion)
	if errors.Is(err, sql.ErrNoRows) {
		var exists bool
		if probeErr := r.db.GetContext(ctx, &exists, `SELECT EXISTS(SELECT 1 FROM students WHERE user_id = $1)`, s.UserID); probeErr != nil {
			return 0, probeErr
		}
		if !exists {
			return 0, management.ErrStudentNotFound
		}
		return 0, management.ErrConflict
	}
	if err != nil {
		return 0, err
	}
	return newVersion, nil
}

// SetStudentStatus transitions the status only when the current status equals
// from — the guard is the WHERE clause, one atomic statement.
func (r *StudentRepository) SetStudentStatus(ctx context.Context, studentID uuid.UUID, from, to management.StudentStatus) (bool, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE students SET status = $3, version = version + 1 WHERE user_id = $1 AND status = $2`,
		studentID, from, to,
	)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// ListCohortYears returns the per-cohort head counts of a program.
func (r *StudentRepository) ListCohortYears(ctx context.Context, programID uuid.UUID) ([]management.CohortYearSummary, error) {
	var summaries []management.CohortYearSummary
	err := r.db.SelectContext(ctx, &summaries, `
		SELECT current_cohort_year AS cohort_year, COUNT(*) AS student_count
		FROM students
		WHERE program_id = $1
		GROUP BY current_cohort_year
		ORDER BY current_cohort_year DESC`, programID)
	return summaries, err
}

// ListCohortHistory returns a student's cohort moves, newest first.
func (r *StudentRepository) ListCohortHistory(ctx context.Context, studentID uuid.UUID) ([]management.CohortHistory, error) {
	var histories []management.CohortHistory
	err := r.db.SelectContext(ctx, &histories, `SELECT * FROM student_cohort_history WHERE student_id = $1 ORDER BY changed_at DESC`, studentID)
	return histories, err
}

// GetTranscriptData assembles the raw enrollment history a transcript is
// built from.
func (r *StudentRepository) GetTranscriptData(ctx context.Context, studentID uuid.UUID) (*management.TranscriptData, error) {
	var data management.TranscriptData

	err := r.db.QueryRowxContext(ctx, `
		SELECT u.full_name_en, p.name_en
		FROM students s
		JOIN users u ON s.user_id = u.id
		JOIN programs p ON s.program_id = p.id
		WHERE s.user_id = $1`, studentID).Scan(&data.StudentName, &data.ProgramName)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, management.ErrStudentNotFound
	}
	if err != nil {
		return nil, err
	}

	rows, err := r.db.QueryxContext(ctx, `
		SELECT ay.year, sem.semester, c.code, c.name_en, c.credits, e.final_grade, e.status
		FROM course_enrollments e
		JOIN course_offerings o ON e.offering_id = o.id
		JOIN courses c ON o.course_id = c.id
		JOIN semesters sem ON o.semester_id = sem.id
		JOIN academic_years ay ON sem.academic_year_id = ay.id
		WHERE e.student_id = $1
		ORDER BY ay.year, sem.semester, c.code`, studentID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var e management.TranscriptEnrollment
		if err := rows.Scan(&e.AcademicYear, &e.Semester, &e.CourseCode, &e.CourseName, &e.Credits, &e.Grade, &e.Status); err != nil {
			return nil, err
		}
		data.Enrollments = append(data.Enrollments, e)
	}
	return &data, rows.Err()
}

// GetStudentScope locates the user's student record in the university
// structure, or nil when the user is not a student.
func (r *StudentRepository) GetStudentScope(ctx context.Context, userID uuid.UUID) (*management.StudentScope, error) {
	var scope management.StudentScope
	err := r.db.GetContext(ctx, &scope, `
		SELECT s.user_id, s.program_id, p.department_id, d.college_id, s.status
		FROM students s
		JOIN programs p ON s.program_id = p.id
		JOIN departments d ON p.department_id = d.id
		WHERE s.user_id = $1`, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &scope, nil
}

// ── Semester-side student reads (management.SemesterStudentProvider) ──────────

// GetActiveStudents returns the active students, optionally narrowed to one
// program and cohort.
func (r *StudentRepository) GetActiveStudents(ctx context.Context, programID *uuid.UUID, cohortYear *int) ([]management.AcademicStudentInfo, error) {
	conditions := []string{"s.status = 'active'"}
	var args []any
	argN := 1

	if programID != nil {
		conditions = append(conditions, fmt.Sprintf("s.program_id = $%d", argN))
		args = append(args, *programID)
		argN++
	}
	if cohortYear != nil {
		conditions = append(conditions, fmt.Sprintf("s.current_cohort_year = $%d", argN))
		args = append(args, *cohortYear)
	}

	query := `
		SELECT s.user_id, u.full_name_en, s.program_id, s.current_cohort_year, s.current_year, s.shift, s.status
		FROM students s
		JOIN users u ON s.user_id = u.id
		WHERE ` + strings.Join(conditions, " AND ")
	return r.scanAcademicStudents(ctx, query, args...)
}

// GetStudentsInSemester returns the students with any enrollment in the
// semester's offerings.
func (r *StudentRepository) GetStudentsInSemester(ctx context.Context, semesterID uuid.UUID) ([]management.AcademicStudentInfo, error) {
	return r.scanAcademicStudents(ctx, `
		SELECT DISTINCT s.user_id, u.full_name_en, s.program_id, s.current_cohort_year, s.current_year, s.shift, s.status
		FROM students s
		JOIN users u ON s.user_id = u.id
		JOIN course_enrollments e ON e.student_id = s.user_id
		JOIN course_offerings o ON e.offering_id = o.id
		WHERE o.semester_id = $1`, semesterID)
}

// UpdateStudentProgression moves a student to a new stage and cohort during
// year-end progression.
func (r *StudentRepository) UpdateStudentProgression(ctx context.Context, studentID uuid.UUID, currentYear, cohortYear int) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE students SET current_year = $2, current_cohort_year = $3, version = version + 1 WHERE user_id = $1`,
		studentID, currentYear, cohortYear)
	return err
}

// RecordCohortChange appends a cohort-history row.
func (r *StudentRepository) RecordCohortChange(ctx context.Context, studentID uuid.UUID, fromCohort, toCohort, fromYear, toYear int, reason management.CohortChangeReason) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO student_cohort_history (student_id, from_cohort_year, to_cohort_year, from_year, to_year, reason)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		studentID, fromCohort, toCohort, fromYear, toYear, reason)
	return err
}

func (r *StudentRepository) scanAcademicStudents(ctx context.Context, query string, args ...any) ([]management.AcademicStudentInfo, error) {
	rows, err := r.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var students []management.AcademicStudentInfo
	for rows.Next() {
		var s management.AcademicStudentInfo
		if err := rows.Scan(&s.UserID, &s.Name, &s.ProgramID, &s.CurrentCohortYear, &s.CurrentYear, &s.Shift, &s.Status); err != nil {
			return nil, err
		}
		students = append(students, s)
	}
	return students, rows.Err()
}

// ── Identity reader adapter ───────────────────────────────────────────────────
