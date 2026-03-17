package student

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/ranjdotdev/e-campus-server/internal/academic"
	"github.com/ranjdotdev/e-campus-server/internal/pagination"
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateStudent(ctx context.Context, s *Student) error {
	query := `
		INSERT INTO students (user_id, program_id, admission_year, current_cohort_year, current_year, shift, tuition, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, enrolled_at, created_at`
	return r.db.QueryRowxContext(ctx, query,
		s.UserID, s.ProgramID, s.AdmissionYear, s.CurrentCohortYear, s.CurrentYear, s.Shift, s.Tuition, s.Status,
	).Scan(&s.ID, &s.EnrolledAt, &s.CreatedAt)
}

func (r *Repository) GetStudent(ctx context.Context, id uuid.UUID) (*Student, error) {
	var s Student
	query := `SELECT * FROM students WHERE id = $1`
	err := r.db.GetContext(ctx, &s, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrStudentNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *Repository) GetStudentByUserID(ctx context.Context, userID uuid.UUID) (*Student, error) {
	var s Student
	query := `SELECT * FROM students WHERE user_id = $1`
	err := r.db.GetContext(ctx, &s, query, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrStudentNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *Repository) ListStudents(ctx context.Context, params pagination.PageParams, filters StudentFilters) ([]Student, bool, error) {
	var students []Student

	var conditions []string
	var args []interface{}
	argIndex := 1

	if filters.ProgramID != nil {
		conditions = append(conditions, fmt.Sprintf("s.program_id = $%d", argIndex))
		args = append(args, *filters.ProgramID)
		argIndex++
	}
	if filters.CohortYear != nil {
		conditions = append(conditions, fmt.Sprintf("s.current_cohort_year = $%d", argIndex))
		args = append(args, *filters.CohortYear)
		argIndex++
	}
	if filters.Stage != nil {
		conditions = append(conditions, fmt.Sprintf("s.current_year = $%d", argIndex))
		args = append(args, *filters.Stage)
		argIndex++
	}
	if filters.Status != nil {
		conditions = append(conditions, fmt.Sprintf("s.status = $%d", argIndex))
		args = append(args, *filters.Status)
		argIndex++
	}
	if filters.Shift != nil {
		conditions = append(conditions, fmt.Sprintf("s.shift = $%d", argIndex))
		args = append(args, *filters.Shift)
		argIndex++
	}
	if filters.Query != nil && *filters.Query != "" {
		conditions = append(conditions, fmt.Sprintf("(u.full_name_en ILIKE $%d OR u.email ILIKE $%d)", argIndex, argIndex))
		args = append(args, "%"+*filters.Query+"%")
		argIndex++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	query := `
		SELECT s.* FROM students s
		JOIN users u ON s.user_id = u.id
		` + whereClause + `
		ORDER BY s.created_at DESC
		LIMIT $` + fmt.Sprintf("%d", argIndex)

	args = append(args, params.Limit+1)

	if err := r.db.SelectContext(ctx, &students, query, args...); err != nil {
		return nil, false, err
	}

	hasMore := len(students) > params.Limit
	if hasMore {
		students = students[:params.Limit]
	}

	return students, hasMore, nil
}

func (r *Repository) UpdateStudent(ctx context.Context, s *Student) error {
	query := `
		UPDATE students
		SET current_cohort_year = $2, current_year = $3, shift = $4, tuition = $5, status = $6
		WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, s.ID, s.CurrentCohortYear, s.CurrentYear, s.Shift, s.Tuition, s.Status)
	return err
}

func (r *Repository) StudentExistsByUserID(ctx context.Context, userID uuid.UUID) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM students WHERE user_id = $1)`
	err := r.db.GetContext(ctx, &exists, query, userID)
	return exists, err
}

func (r *Repository) CreateLeave(ctx context.Context, l *Leave) error {
	query := `
		INSERT INTO student_leaves (student_id, type, academic_year_id, reason, start_date, end_date, approved_by, approved_at, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at`
	return r.db.QueryRowxContext(ctx, query,
		l.StudentID, l.Type, l.AcademicYearID, l.Reason, l.StartDate, l.EndDate, l.ApprovedBy, l.ApprovedAt, l.Notes,
	).Scan(&l.ID, &l.CreatedAt)
}

func (r *Repository) GetLeave(ctx context.Context, id uuid.UUID) (*Leave, error) {
	var l Leave
	query := `SELECT * FROM student_leaves WHERE id = $1`
	err := r.db.GetContext(ctx, &l, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrLeaveNotFound
	}
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func (r *Repository) ListLeaves(ctx context.Context, studentID uuid.UUID) ([]Leave, error) {
	var leaves []Leave
	query := `SELECT * FROM student_leaves WHERE student_id = $1 ORDER BY start_date DESC`
	if err := r.db.SelectContext(ctx, &leaves, query, studentID); err != nil {
		return nil, err
	}
	return leaves, nil
}

func (r *Repository) UpdateLeave(ctx context.Context, l *Leave) error {
	query := `
		UPDATE student_leaves
		SET end_date = $2, approved_by = $3, approved_at = $4, notes = $5
		WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, l.ID, l.EndDate, l.ApprovedBy, l.ApprovedAt, l.Notes)
	return err
}

func (r *Repository) GetActiveLeave(ctx context.Context, studentID uuid.UUID) (*Leave, error) {
	var l Leave
	query := `SELECT * FROM student_leaves WHERE student_id = $1 AND end_date IS NULL ORDER BY created_at DESC LIMIT 1`
	err := r.db.GetContext(ctx, &l, query, studentID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func (r *Repository) AddLeaveSemesters(ctx context.Context, leaveID uuid.UUID, semesterIDs []uuid.UUID) error {
	if len(semesterIDs) == 0 {
		return nil
	}
	query := `INSERT INTO leave_semesters (leave_id, semester_id) VALUES ($1, $2)`
	for _, semID := range semesterIDs {
		if _, err := r.db.ExecContext(ctx, query, leaveID, semID); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) GetLeaveSemesters(ctx context.Context, leaveID uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	query := `SELECT semester_id FROM leave_semesters WHERE leave_id = $1`
	err := r.db.SelectContext(ctx, &ids, query, leaveID)
	return ids, err
}

func (r *Repository) CreateCohortHistory(ctx context.Context, h *CohortHistory) error {
	query := `
		INSERT INTO student_cohort_history (student_id, from_cohort_year, to_cohort_year, from_year, to_year, reason, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, changed_at`
	return r.db.QueryRowxContext(ctx, query,
		h.StudentID, h.FromCohortYear, h.ToCohortYear, h.FromYear, h.ToYear, h.Reason, h.Notes,
	).Scan(&h.ID, &h.ChangedAt)
}

func (r *Repository) ListCohortHistory(ctx context.Context, studentID uuid.UUID) ([]CohortHistory, error) {
	var histories []CohortHistory
	query := `SELECT * FROM student_cohort_history WHERE student_id = $1 ORDER BY changed_at DESC`
	if err := r.db.SelectContext(ctx, &histories, query, studentID); err != nil {
		return nil, err
	}
	return histories, nil
}

func (r *Repository) GetTranscriptData(ctx context.Context, studentID uuid.UUID) (*TranscriptData, error) {
	var data TranscriptData

	studentQuery := `
		SELECT u.full_name_en, p.name_en
		FROM students s
		JOIN users u ON s.user_id = u.id
		JOIN programs p ON s.program_id = p.id
		WHERE s.id = $1`

	var studentName, programName string
	if err := r.db.QueryRowxContext(ctx, studentQuery, studentID).Scan(&studentName, &programName); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrStudentNotFound
		}
		return nil, err
	}
	data.StudentName = studentName
	data.ProgramName = programName

	enrollmentQuery := `
		SELECT ay.year, sem.semester, c.code, c.name_en, c.credits, e.final_grade, e.status
		FROM course_enrollments e
		JOIN students s ON e.student_id = s.user_id
		JOIN course_offerings o ON e.offering_id = o.id
		JOIN courses c ON o.course_id = c.id
		JOIN semesters sem ON o.semester_id = sem.id
		JOIN academic_years ay ON sem.academic_year_id = ay.id
		WHERE s.id = $1
		ORDER BY ay.year, sem.semester, c.code`

	rows, err := r.db.QueryxContext(ctx, enrollmentQuery, studentID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var e EnrollmentData
		if err := rows.Scan(&e.AcademicYear, &e.Semester, &e.CourseCode, &e.CourseName, &e.Credits, &e.Grade, &e.Status); err != nil {
			return nil, err
		}
		data.Enrollments = append(data.Enrollments, e)
	}

	return &data, nil
}

func (r *Repository) GetActiveStudentsForAcademic(ctx context.Context, programID *uuid.UUID, cohortYear *int) ([]academic.StudentInfo, error) {
	var conditions []string
	var args []interface{}
	argIndex := 1

	conditions = append(conditions, "s.status = 'active'")

	if programID != nil {
		conditions = append(conditions, fmt.Sprintf("s.program_id = $%d", argIndex))
		args = append(args, *programID)
		argIndex++
	}
	if cohortYear != nil {
		conditions = append(conditions, fmt.Sprintf("s.current_cohort_year = $%d", argIndex))
		args = append(args, *cohortYear)
	}

	query := `
		SELECT s.id, s.user_id, u.full_name_en, s.program_id, s.current_cohort_year, s.current_year, s.shift, s.status
		FROM students s
		JOIN users u ON s.user_id = u.id
		WHERE ` + strings.Join(conditions, " AND ")

	rows, err := r.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var students []academic.StudentInfo
	for rows.Next() {
		var s academic.StudentInfo
		if err := rows.Scan(&s.ID, &s.UserID, &s.Name, &s.ProgramID, &s.CurrentCohortYear, &s.CurrentYear, &s.Shift, &s.Status); err != nil {
			return nil, err
		}
		students = append(students, s)
	}
	return students, nil
}

func (r *Repository) GetStudentsByProgramForAcademic(ctx context.Context, programID uuid.UUID) ([]academic.StudentInfo, error) {
	query := `
		SELECT s.id, s.user_id, u.full_name_en, s.program_id, s.current_cohort_year, s.current_year, s.shift, s.status
		FROM students s
		JOIN users u ON s.user_id = u.id
		WHERE s.program_id = $1`

	rows, err := r.db.QueryxContext(ctx, query, programID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var students []academic.StudentInfo
	for rows.Next() {
		var s academic.StudentInfo
		if err := rows.Scan(&s.ID, &s.UserID, &s.Name, &s.ProgramID, &s.CurrentCohortYear, &s.CurrentYear, &s.Shift, &s.Status); err != nil {
			return nil, err
		}
		students = append(students, s)
	}
	return students, nil
}

func (r *Repository) GetStudentsInSemesterForAcademic(ctx context.Context, semesterID uuid.UUID) ([]academic.StudentInfo, error) {
	query := `
		SELECT DISTINCT s.id, s.user_id, u.full_name_en, s.program_id, s.current_cohort_year, s.current_year, s.shift, s.status
		FROM students s
		JOIN users u ON s.user_id = u.id
		JOIN course_enrollments e ON e.student_id = s.user_id
		JOIN course_offerings o ON e.offering_id = o.id
		WHERE o.semester_id = $1`

	rows, err := r.db.QueryxContext(ctx, query, semesterID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var students []academic.StudentInfo
	for rows.Next() {
		var s academic.StudentInfo
		if err := rows.Scan(&s.ID, &s.UserID, &s.Name, &s.ProgramID, &s.CurrentCohortYear, &s.CurrentYear, &s.Shift, &s.Status); err != nil {
			return nil, err
		}
		students = append(students, s)
	}
	return students, nil
}

func (r *Repository) UpdateStudentProgression(ctx context.Context, studentID uuid.UUID, currentYear, cohortYear int) error {
	query := `UPDATE students SET current_year = $2, current_cohort_year = $3 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, studentID, currentYear, cohortYear)
	return err
}

func (r *Repository) RecordCohortChangeForAcademic(ctx context.Context, studentID uuid.UUID, fromCohort, toCohort, fromYear, toYear int, reason string) error {
	query := `
		INSERT INTO student_cohort_history (student_id, from_cohort_year, to_cohort_year, from_year, to_year, reason)
		VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.db.ExecContext(ctx, query, studentID, fromCohort, toCohort, fromYear, toYear, reason)
	return err
}
