package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/randdotdev/e-campus-server/internal/classroom"
)

// Readers satisfies classroom's management-facing reader ports with the
// read-only published-table lookups §19a sanctions for classroom: single
// indexed queries against course_offerings, courses, semesters, students,
// and course_enrollments. Writes into those tables stay with management.
type Readers struct {
	db *sqlx.DB
}

func NewReaders(db *sqlx.DB) *Readers {
	return &Readers{db: db}
}

var (
	_ classroom.OfferingReader   = (*Readers)(nil)
	_ classroom.StudentReader    = (*Readers)(nil)
	_ classroom.EnrollmentReader = (*Readers)(nil)
	_ classroom.UserReader       = (*Readers)(nil)
)

func (r *Readers) CourseCodeByOffering(ctx context.Context, offeringID uuid.UUID) (string, error) {
	var code string
	err := r.db.GetContext(ctx, &code, `
		SELECT c.code FROM course_offerings co
		JOIN courses c ON c.id = co.course_id
		WHERE co.id = $1`, offeringID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", classroom.ErrInvalidInput
	}
	return code, err
}

func (r *Readers) SemesterStatus(ctx context.Context, offeringID uuid.UUID) (string, error) {
	var status string
	err := r.db.GetContext(ctx, &status, `
		SELECT s.status FROM semesters s
		JOIN course_offerings co ON co.semester_id = s.id
		WHERE co.id = $1`, offeringID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", classroom.ErrInvalidInput
	}
	return status, err
}

func (r *Readers) PassThreshold(ctx context.Context, offeringID uuid.UUID) (int, error) {
	var threshold int
	err := r.db.GetContext(ctx, &threshold, `
		SELECT s.pass_threshold FROM semesters s
		JOIN course_offerings co ON co.semester_id = s.id
		WHERE co.id = $1`, offeringID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, classroom.ErrInvalidInput
	}
	return threshold, err
}

func (r *Readers) StudentProgramCohort(ctx context.Context, userID uuid.UUID) (uuid.UUID, int, error) {
	var row struct {
		ProgramID  uuid.UUID `db:"program_id"`
		CohortYear int       `db:"current_cohort_year"`
	}
	err := r.db.GetContext(ctx, &row,
		`SELECT program_id, current_cohort_year FROM students WHERE user_id = $1`, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, 0, classroom.ErrNotStudent
	}
	return row.ProgramID, row.CohortYear, err
}

// EnrolledUserIDs reads the roster; course_enrollments.student_id holds
// account IDs (users.id) by the schema's FK.
func (r *Readers) EnrolledUserIDs(ctx context.Context, offeringID uuid.UUID) ([]uuid.UUID, error) {
	ids := []uuid.UUID{}
	err := r.db.SelectContext(ctx, &ids, `
		SELECT student_id FROM course_enrollments
		WHERE offering_id = $1 AND status = 'enrolled'`, offeringID)
	return ids, err
}

func (r *Readers) AllEnrolled(ctx context.Context, offeringID uuid.UUID, userIDs []uuid.UUID) (bool, error) {
	if len(userIDs) == 0 {
		return false, nil
	}
	var count int
	err := r.db.GetContext(ctx, &count, `
		SELECT COUNT(DISTINCT student_id) FROM course_enrollments
		WHERE offering_id = $1 AND status = 'enrolled' AND student_id = ANY($2)`,
		offeringID, pq.Array(userIDs))
	if err != nil {
		return false, err
	}
	return count == len(userIDs), nil
}

// UserName reads a user's display name (users is published to every
// context's lists).
func (r *Readers) UserName(ctx context.Context, userID uuid.UUID) (string, error) {
	var name string
	err := r.db.GetContext(ctx, &name, `SELECT full_name_en FROM users WHERE id = $1`, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", classroom.ErrInvalidInput
	}
	return name, err
}
