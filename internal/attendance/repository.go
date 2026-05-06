package attendance

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) InitializeAttendance(ctx context.Context, lessonID uuid.UUID, studentIDs []uuid.UUID) (int, error) {
	if len(studentIDs) == 0 {
		return 0, nil
	}

	query := `
		INSERT INTO attendance (lesson_id, student_id)
		SELECT $1, unnest($2::uuid[])
		ON CONFLICT (lesson_id, student_id) DO NOTHING`

	result, err := r.db.ExecContext(ctx, query, lessonID, studentIDs)
	if err != nil {
		return 0, err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return int(rows), nil
}

func (r *Repository) BulkUpdateAttendance(ctx context.Context, lessonID uuid.UUID, markerID uuid.UUID, records []AttendanceUpdate) error {
	if len(records) == 0 {
		return nil
	}

	now := time.Now()
	args := []any{markerID, now, lessonID}
	var cases, ids []string

	for _, r := range records {
		argIdx := len(args) + 1
		cases = append(cases, fmt.Sprintf("WHEN $%d THEN $%d", argIdx, argIdx+1))
		ids = append(ids, fmt.Sprintf("$%d", argIdx))
		args = append(args, r.ID, r.Percentage)
	}

	query := fmt.Sprintf(
		"UPDATE attendance SET percentage = CASE id %s END, marked_by = $1, marked_at = $2 WHERE lesson_id = $3 AND id IN (%s)",
		strings.Join(cases, " "),
		strings.Join(ids, ", "),
	)

	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *Repository) UpdateAttendance(ctx context.Context, a *Attendance) error {
	query := `UPDATE attendance SET percentage = $1, marked_by = $2, marked_at = $3 WHERE id = $4`
	_, err := r.db.ExecContext(ctx, query, a.Percentage, a.MarkedBy, a.MarkedAt, a.ID)
	return err
}

func (r *Repository) GetAttendanceByID(ctx context.Context, id uuid.UUID) (*Attendance, error) {
	var a Attendance
	query := `SELECT id, lesson_id, student_id, percentage, marked_by, marked_at, created_at FROM attendance WHERE id = $1`
	if err := r.db.GetContext(ctx, &a, query, id); err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *Repository) GetLessonAttendance(ctx context.Context, lessonID uuid.UUID) ([]AttendanceRecord, error) {
	query := `
		SELECT
			a.id, a.lesson_id, a.student_id, a.percentage, a.marked_by, a.marked_at, a.created_at,
			u.full_name_en as student_name,
			e.status as excuse_status,
			e.reason as excuse_reason
		FROM attendance a
		JOIN users u ON a.student_id = u.id
		LEFT JOIN excuse_requests e ON a.lesson_id = e.lesson_id AND a.student_id = e.student_id
		WHERE a.lesson_id = $1
		ORDER BY u.full_name_en`
	var records []AttendanceRecord
	if err := r.db.SelectContext(ctx, &records, query, lessonID); err != nil {
		return nil, err
	}
	return records, nil
}

func (r *Repository) GetOfferingAttendance(ctx context.Context, offeringID uuid.UUID) ([]AttendanceRecord, error) {
	query := `
		SELECT
			a.id, a.lesson_id, a.student_id, a.percentage, a.marked_by, a.marked_at, a.created_at,
			u.full_name_en as student_name,
			e.status as excuse_status,
			e.reason as excuse_reason
		FROM attendance a
		JOIN users u ON a.student_id = u.id
		JOIN lessons l ON a.lesson_id = l.id
		JOIN sections s ON l.section_id = s.id
		LEFT JOIN excuse_requests e ON a.lesson_id = e.lesson_id AND a.student_id = e.student_id
		WHERE s.offering_id = $1
		ORDER BY l.order_index, u.full_name_en`
	var records []AttendanceRecord
	if err := r.db.SelectContext(ctx, &records, query, offeringID); err != nil {
		return nil, err
	}
	return records, nil
}

func (r *Repository) GetAttendanceSummaries(ctx context.Context, offeringID uuid.UUID) ([]AttendanceSummary, error) {
	query := `
		WITH lesson_data AS (
			SELECT
				l.id as lesson_id,
				COALESCE(ls.duration_hours, 1) as duration_hours
			FROM lessons l
			JOIN sections s ON l.section_id = s.id
			LEFT JOIN lesson_schedules ls ON l.id = ls.lesson_id
			WHERE s.offering_id = $1
				AND l.attendance_required = true
				AND ls.scheduled_at < NOW()
		),
		student_attendance AS (
			SELECT
				e.student_id,
				u.full_name_en as student_name,
				COALESCE(SUM(ld.duration_hours), 0) as total_hours,
				COALESCE(SUM(
					CASE WHEN a.marked_by IS NOT NULL
					THEN ld.duration_hours * a.percentage / 100.0
					ELSE 0 END
				), 0) as attended_hours,
				COALESCE(SUM(
					CASE WHEN er.status = 'approved'
					THEN ld.duration_hours
					ELSE 0 END
				), 0) as excused_hours
			FROM course_enrollments e
			JOIN users u ON e.student_id = u.id
			LEFT JOIN lesson_data ld ON true
			LEFT JOIN attendance a ON a.lesson_id = ld.lesson_id AND a.student_id = e.student_id
			LEFT JOIN excuse_requests er ON er.lesson_id = ld.lesson_id AND er.student_id = e.student_id
			WHERE e.offering_id = $1
			GROUP BY e.student_id, u.full_name
		)
		SELECT student_id, student_name, total_hours, attended_hours, excused_hours
		FROM student_attendance
		ORDER BY student_name`
	var summaries []AttendanceSummary
	if err := r.db.SelectContext(ctx, &summaries, query, offeringID); err != nil {
		return nil, err
	}
	return summaries, nil
}

func (r *Repository) GetStudentAttendance(ctx context.Context, studentID, offeringID uuid.UUID) ([]StudentAttendance, error) {
	query := `
		SELECT
			l.id as lesson_id,
			l.title as lesson_title,
			s.title as section_title,
			ls.scheduled_at,
			ls.duration_hours,
			a.percentage,
			a.marked_by,
			e.status as excuse_status
		FROM lessons l
		JOIN sections s ON l.section_id = s.id
		LEFT JOIN lesson_schedules ls ON l.id = ls.lesson_id
		LEFT JOIN attendance a ON l.id = a.lesson_id AND a.student_id = $1
		LEFT JOIN excuse_requests e ON l.id = e.lesson_id AND e.student_id = $1
		WHERE s.offering_id = $2 AND l.attendance_required = true
		ORDER BY s.order_index, l.order_index`
	var records []StudentAttendance
	if err := r.db.SelectContext(ctx, &records, query, studentID, offeringID); err != nil {
		return nil, err
	}
	return records, nil
}

func (r *Repository) GetStudentCourseAttendances(ctx context.Context, studentID uuid.UUID) ([]CourseAttendance, error) {
	query := `
		WITH course_data AS (
			SELECT
				o.id as offering_id,
				c.name as course_name,
				c.code as course_code,
				COUNT(l.id) as total_lessons,
				COUNT(a.id) FILTER (WHERE a.marked_by IS NOT NULL AND a.percentage > 0) as attended_count,
				COUNT(a.id) FILTER (WHERE a.marked_by IS NOT NULL AND a.percentage = 0 AND er.status IS DISTINCT FROM 'approved') as absent_count,
				COUNT(er.id) FILTER (WHERE er.status = 'approved') as excused_count
			FROM course_enrollments e
			JOIN course_offerings o ON e.offering_id = o.id
			JOIN courses c ON o.course_id = c.id
			JOIN sections s ON s.offering_id = o.id
			JOIN lessons l ON l.section_id = s.id AND l.attendance_required = true
			LEFT JOIN attendance a ON a.lesson_id = l.id AND a.student_id = e.student_id
			LEFT JOIN excuse_requests er ON er.lesson_id = l.id AND er.student_id = e.student_id
			WHERE e.student_id = $1
			GROUP BY o.id, c.name, c.code
		)
		SELECT offering_id, course_name, course_code, total_lessons, attended_count, absent_count, excused_count
		FROM course_data
		ORDER BY course_name`
	var courses []CourseAttendance
	if err := r.db.SelectContext(ctx, &courses, query, studentID); err != nil {
		return nil, err
	}
	return courses, nil
}

func (r *Repository) CreateExcuseRequest(ctx context.Context, e *ExcuseRequest) error {
	query := `
		INSERT INTO excuse_requests (lesson_id, student_id, reason, status, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`
	return r.db.QueryRowxContext(ctx, query,
		e.LessonID, e.StudentID, e.Reason, e.Status, e.CreatedAt,
	).Scan(&e.ID)
}

func (r *Repository) UpdateExcuseRequest(ctx context.Context, e *ExcuseRequest) error {
	query := `UPDATE excuse_requests SET status = $1, note = $2, reviewed_by = $3, reviewed_at = $4 WHERE id = $5`
	_, err := r.db.ExecContext(ctx, query, e.Status, e.Note, e.ReviewedBy, e.ReviewedAt, e.ID)
	return err
}

func (r *Repository) GetExcuseRequestByID(ctx context.Context, id uuid.UUID) (*ExcuseRequest, error) {
	var e ExcuseRequest
	query := `SELECT id, lesson_id, student_id, reason, status, note, reviewed_by, reviewed_at, created_at FROM excuse_requests WHERE id = $1`
	if err := r.db.GetContext(ctx, &e, query, id); err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *Repository) GetExcuseByLessonAndStudent(ctx context.Context, lessonID, studentID uuid.UUID) (*ExcuseRequest, error) {
	var e ExcuseRequest
	query := `SELECT id, lesson_id, student_id, reason, status, note, reviewed_by, reviewed_at, created_at FROM excuse_requests WHERE lesson_id = $1 AND student_id = $2`
	if err := r.db.GetContext(ctx, &e, query, lessonID, studentID); err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *Repository) GetPendingExcuses(ctx context.Context, offeringID uuid.UUID) ([]ExcuseRequest, error) {
	query := `
		SELECT e.id, e.lesson_id, e.student_id, e.reason, e.status, e.note, e.reviewed_by, e.reviewed_at, e.created_at
		FROM excuse_requests e
		JOIN lessons l ON e.lesson_id = l.id
		JOIN sections s ON l.section_id = s.id
		WHERE s.offering_id = $1 AND e.status = 'pending'
		ORDER BY e.created_at`
	var excuses []ExcuseRequest
	if err := r.db.SelectContext(ctx, &excuses, query, offeringID); err != nil {
		return nil, err
	}
	return excuses, nil
}
