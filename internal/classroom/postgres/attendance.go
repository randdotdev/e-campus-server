package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/randdotdev/e-campus-server/internal/classroom"
)

// AttendanceRepository is the SQL adapter for attendance and excuses.
// Lesson scoping goes through sections, as everywhere in this context.
type AttendanceRepository struct {
	db *sqlx.DB
}

func NewAttendanceRepository(db *sqlx.DB) *AttendanceRepository {
	return &AttendanceRepository{db: db}
}

var (
	_ classroom.AttendanceRepository = (*AttendanceRepository)(nil)
	_ classroom.AttendanceRateReader = (*AttendanceRepository)(nil)
)

func (r *AttendanceRepository) LessonForAttendance(ctx context.Context, offeringID, lessonID uuid.UUID) (bool, error) {
	var required bool
	err := r.db.GetContext(ctx, &required, `
		SELECT l.attendance_required FROM lessons l
		JOIN sections s ON s.id = l.section_id
		WHERE l.id = $1 AND s.offering_id = $2`, lessonID, offeringID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, classroom.ErrLessonNotFound
	}
	return required, err
}

// InitializeAttendance inserts a zero row per student; ON CONFLICT keeps
// the call idempotent when the sheet partially exists.
func (r *AttendanceRepository) InitializeAttendance(ctx context.Context, lessonID uuid.UUID, studentIDs []uuid.UUID) (int, error) {
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO attendance (lesson_id, student_id, percentage)
		SELECT $1, unnest($2::uuid[]), 0
		ON CONFLICT (lesson_id, student_id) DO NOTHING`,
		lessonID, pq.Array(studentIDs))
	if err != nil {
		return 0, err
	}
	n, _ := result.RowsAffected()
	return int(n), nil
}

func (r *AttendanceRepository) GetAttendance(ctx context.Context, offeringID, id uuid.UUID) (*classroom.Attendance, error) {
	var a classroom.Attendance
	err := r.db.GetContext(ctx, &a, `
		SELECT a.* FROM attendance a
		JOIN lessons l ON l.id = a.lesson_id
		JOIN sections s ON s.id = l.section_id
		WHERE a.id = $1 AND s.offering_id = $2`, id, offeringID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, classroom.ErrAttendanceNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *AttendanceRepository) MarkAttendance(ctx context.Context, id, markerID uuid.UUID, percentage int, at time.Time) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE attendance SET percentage = $1, marked_by = $2, marked_at = $3
		WHERE id = $4`, percentage, markerID, at, id)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return classroom.ErrAttendanceNotFound
	}
	return nil
}

func (r *AttendanceRepository) BulkMark(ctx context.Context, lessonID, markerID uuid.UUID, updates []classroom.AttendanceUpdate, at time.Time) error {
	return inTx(ctx, r.db, func(tx *sqlx.Tx) error {
		for _, u := range updates {
			result, err := tx.ExecContext(ctx, `
				UPDATE attendance SET percentage = $1, marked_by = $2, marked_at = $3
				WHERE id = $4 AND lesson_id = $5`,
				u.Percentage, markerID, at, u.AttendanceID, lessonID)
			if err != nil {
				return err
			}
			if n, _ := result.RowsAffected(); n == 0 {
				return classroom.ErrAttendanceNotFound
			}
		}
		return nil
	})
}

const attendanceRecordQuery = `
	SELECT a.*, u.full_name_en AS student_name, u.username AS student_username,
	       er.status AS excuse_status
	FROM attendance a
	JOIN users u ON u.id = a.student_id
	LEFT JOIN excuse_requests er ON er.lesson_id = a.lesson_id AND er.student_id = a.student_id`

func (r *AttendanceRepository) ListLessonAttendance(ctx context.Context, lessonID uuid.UUID) ([]classroom.AttendanceRecord, error) {
	records := []classroom.AttendanceRecord{}
	err := r.db.SelectContext(ctx, &records,
		attendanceRecordQuery+` WHERE a.lesson_id = $1 ORDER BY u.full_name_en`, lessonID)
	return records, err
}

func (r *AttendanceRepository) ListOfferingAttendance(ctx context.Context, offeringID uuid.UUID) ([]classroom.AttendanceRecord, error) {
	records := []classroom.AttendanceRecord{}
	err := r.db.SelectContext(ctx, &records, attendanceRecordQuery+`
		JOIN lessons l ON l.id = a.lesson_id
		JOIN sections s ON s.id = l.section_id
		WHERE s.offering_id = $1
		ORDER BY a.created_at DESC`, offeringID)
	return records, err
}

func (r *AttendanceRepository) ListSummaries(ctx context.Context, offeringID uuid.UUID) ([]classroom.AttendanceSummary, error) {
	summaries := []classroom.AttendanceSummary{}
	err := r.db.SelectContext(ctx, &summaries, `
		SELECT a.student_id, u.full_name_en AS student_name,
		       COALESCE(SUM(COALESCE(l.duration_hours, 1)), 0) AS total_hours,
		       COALESCE(SUM(COALESCE(l.duration_hours, 1) * a.percentage / 100.0)
		           FILTER (WHERE er.status IS DISTINCT FROM 'approved'), 0) AS attended_hours,
		       COALESCE(SUM(COALESCE(l.duration_hours, 1))
		           FILTER (WHERE er.status = 'approved'), 0) AS excused_hours
		FROM attendance a
		JOIN lessons l ON l.id = a.lesson_id
		JOIN sections s ON s.id = l.section_id
		JOIN users u ON u.id = a.student_id
		LEFT JOIN excuse_requests er ON er.lesson_id = a.lesson_id AND er.student_id = a.student_id
		WHERE s.offering_id = $1
		GROUP BY a.student_id, u.full_name_en
		ORDER BY u.full_name_en`, offeringID)
	return summaries, err
}

func (r *AttendanceRepository) ListStudentAttendance(ctx context.Context, offeringID, studentID uuid.UUID) ([]classroom.StudentLessonAttendance, error) {
	rows := []classroom.StudentLessonAttendance{}
	err := r.db.SelectContext(ctx, &rows, `
		SELECT l.id AS lesson_id, l.title AS lesson_title,
		       COALESCE(a.percentage, 0) AS percentage,
		       (a.marked_by IS NOT NULL) AS marked,
		       er.status AS excuse_status
		FROM lessons l
		JOIN sections s ON s.id = l.section_id
		LEFT JOIN attendance a ON a.lesson_id = l.id AND a.student_id = $2
		LEFT JOIN excuse_requests er ON er.lesson_id = l.id AND er.student_id = $2
		WHERE s.offering_id = $1 AND l.attendance_required
		ORDER BY l.order_index`, offeringID, studentID)
	return rows, err
}

func (r *AttendanceRepository) CreateExcuse(ctx context.Context, e *classroom.ExcuseRequest) error {
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO excuse_requests (id, lesson_id, student_id, reason, status, created_at)
		VALUES (:id, :lesson_id, :student_id, :reason, :status, :created_at)`, e)
	if isUniqueViolation(err) {
		return classroom.ErrExcuseExists
	}
	return err
}

func (r *AttendanceRepository) GetExcuse(ctx context.Context, offeringID, id uuid.UUID) (*classroom.ExcuseRequest, error) {
	var e classroom.ExcuseRequest
	err := r.db.GetContext(ctx, &e, `
		SELECT er.* FROM excuse_requests er
		JOIN lessons l ON l.id = er.lesson_id
		JOIN sections s ON s.id = l.section_id
		WHERE er.id = $1 AND s.offering_id = $2`, id, offeringID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, classroom.ErrExcuseNotFound
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// ReviewExcuse decides a pending excuse; the pending guard is the WHERE.
func (r *AttendanceRepository) ReviewExcuse(ctx context.Context, id, reviewerID uuid.UUID, status classroom.ExcuseStatus, note *string, at time.Time) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE excuse_requests
		SET status = $1, note = $2, reviewed_by = $3, reviewed_at = $4
		WHERE id = $5 AND status = 'pending'`,
		status, note, reviewerID, at, id)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return classroom.ErrExcuseReviewed
	}
	return nil
}

func (r *AttendanceRepository) ListPendingExcuses(ctx context.Context, offeringID uuid.UUID) ([]classroom.ExcuseWithStudent, error) {
	excuses := []classroom.ExcuseWithStudent{}
	err := r.db.SelectContext(ctx, &excuses, `
		SELECT er.*, u.full_name_en AS student_name, u.username AS student_username,
		       l.title AS lesson_title
		FROM excuse_requests er
		JOIN lessons l ON l.id = er.lesson_id
		JOIN sections s ON s.id = l.section_id
		JOIN users u ON u.id = er.student_id
		WHERE s.offering_id = $1 AND er.status = 'pending'
		ORDER BY er.created_at`, offeringID)
	return excuses, err
}

func (r *AttendanceRepository) ListStudentExcuses(ctx context.Context, offeringID, studentID uuid.UUID) ([]classroom.ExcuseRequest, error) {
	excuses := []classroom.ExcuseRequest{}
	err := r.db.SelectContext(ctx, &excuses, `
		SELECT er.* FROM excuse_requests er
		JOIN lessons l ON l.id = er.lesson_id
		JOIN sections s ON s.id = l.section_id
		WHERE s.offering_id = $1 AND er.student_id = $2
		ORDER BY er.created_at DESC`, offeringID, studentID)
	return excuses, err
}

// StudentAttendanceRate is the grading read: duration-weighted, approved
// excuses out of the denominator, an empty denominator scoring 100.
func (r *AttendanceRepository) StudentAttendanceRate(ctx context.Context, offeringID, studentID uuid.UUID) (float64, error) {
	var rate float64
	err := r.db.GetContext(ctx, &rate, `
		SELECT COALESCE(
			SUM(COALESCE(l.duration_hours, 1) * COALESCE(a.percentage, 0) / 100.0)
				FILTER (WHERE er.status IS DISTINCT FROM 'approved')
			/ NULLIF(SUM(COALESCE(l.duration_hours, 1))
				FILTER (WHERE er.status IS DISTINCT FROM 'approved'), 0)
			* 100, 100)
		FROM lessons l
		JOIN sections s ON s.id = l.section_id
		LEFT JOIN attendance a ON a.lesson_id = l.id AND a.student_id = $2
		LEFT JOIN excuse_requests er ON er.lesson_id = l.id AND er.student_id = $2
		WHERE s.offering_id = $1 AND l.attendance_required`, offeringID, studentID)
	return rate, err
}

// ListCourseAttendance aggregates the student's duration-weighted hours per
// enrolled offering, over attendance-required lessons.
func (r *AttendanceRepository) ListCourseAttendance(ctx context.Context, studentID uuid.UUID) ([]classroom.CourseAttendance, error) {
	rows := []classroom.CourseAttendance{}
	err := r.db.SelectContext(ctx, &rows, `
		SELECT co.id AS offering_id, c.code AS course_code, c.name_en AS course_name,
		       COALESCE(SUM(COALESCE(l.duration_hours, 1)), 0) AS total_hours,
		       COALESCE(SUM(COALESCE(l.duration_hours, 1) * COALESCE(a.percentage, 0) / 100.0)
		           FILTER (WHERE er.status IS DISTINCT FROM 'approved'), 0) AS attended_hours,
		       COALESCE(SUM(COALESCE(l.duration_hours, 1))
		           FILTER (WHERE er.status = 'approved'), 0) AS excused_hours
		FROM course_enrollments e
		JOIN course_offerings co ON co.id = e.offering_id
		JOIN courses c ON c.id = co.course_id
		JOIN sections s ON s.offering_id = co.id
		JOIN lessons l ON l.section_id = s.id AND l.attendance_required
		LEFT JOIN attendance a ON a.lesson_id = l.id AND a.student_id = $1
		LEFT JOIN excuse_requests er ON er.lesson_id = l.id AND er.student_id = $1
		WHERE e.student_id = $1 AND e.status = 'enrolled'
		GROUP BY co.id, c.code, c.name_en
		ORDER BY c.code`, studentID)
	return rows, err
}
