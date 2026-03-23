package grading

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Repository struct {
	db *sqlx.DB
}

var (
	_ GradingRepository       = (*Repository)(nil)
	_ OfferingProvider        = (*Repository)(nil)
	_ ExamScoreProvider       = (*Repository)(nil)
	_ AssignmentScoreProvider = (*Repository)(nil)
	_ AttendanceProvider      = (*Repository)(nil)
	_ EnrollmentProvider      = (*Repository)(nil)
)

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateRules(ctx context.Context, rules *GradingRules) error {
	rules.ID = uuid.New()
	rules.CreatedAt = time.Now()
	rules.UpdatedAt = time.Now()

	query := `
		INSERT INTO grading_rules (id, offering_id, rules, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := r.db.ExecContext(ctx, query,
		rules.ID, rules.OfferingID, rules.Rules, rules.CreatedBy, rules.CreatedAt, rules.UpdatedAt)
	return err
}

func (r *Repository) GetRules(ctx context.Context, offeringID uuid.UUID) (*GradingRules, error) {
	query := `SELECT * FROM grading_rules WHERE offering_id = $1`

	var rules GradingRules
	if err := r.db.GetContext(ctx, &rules, query, offeringID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &rules, nil
}

func (r *Repository) UpdateRules(ctx context.Context, rules *GradingRules) error {
	query := `UPDATE grading_rules SET rules = $1, updated_at = $2 WHERE offering_id = $3`
	_, err := r.db.ExecContext(ctx, query, rules.Rules, rules.UpdatedAt, rules.OfferingID)
	return err
}

func (r *Repository) DeleteRules(ctx context.Context, offeringID uuid.UUID) error {
	query := `DELETE FROM grading_rules WHERE offering_id = $1`
	_, err := r.db.ExecContext(ctx, query, offeringID)
	return err
}

func (r *Repository) GetStudentGrades(ctx context.Context, offeringID uuid.UUID) ([]StudentGrade, error) {
	query := `
		SELECT
			ce.student_id,
			u.full_name_en AS student_name,
			ce.final_grade,
			ce.status,
			ce.completed_at
		FROM course_enrollments ce
		JOIN users u ON u.id = ce.student_id
		WHERE ce.offering_id = $1
			AND ce.status != 'dropped'
		ORDER BY u.full_name_en`

	var grades []StudentGrade
	if err := r.db.SelectContext(ctx, &grades, query, offeringID); err != nil {
		return nil, err
	}
	return grades, nil
}

func (r *Repository) UpdateEnrollmentGrade(ctx context.Context, offeringID, studentID uuid.UUID, grade float64, status string) error {
	query := `
		UPDATE course_enrollments
		SET final_grade = $1, status = $2, completed_at = NOW()
		WHERE offering_id = $3 AND student_id = $4`

	result, err := r.db.ExecContext(ctx, query, grade, status, offeringID, studentID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrStudentNotEnrolled
	}
	return nil
}

func (r *Repository) ClearEnrollmentGrades(ctx context.Context, offeringID uuid.UUID) error {
	query := `
		UPDATE course_enrollments
		SET final_grade = NULL, status = 'enrolled', completed_at = NULL
		WHERE offering_id = $1 AND status IN ('completed', 'failed')`

	_, err := r.db.ExecContext(ctx, query, offeringID)
	return err
}

func (r *Repository) OfferingExists(ctx context.Context, id uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM course_offerings WHERE id = $1)`
	var exists bool
	err := r.db.GetContext(ctx, &exists, query, id)
	return exists, err
}

func (r *Repository) GetSemesterStatus(ctx context.Context, offeringID uuid.UUID) (string, error) {
	query := `
		SELECT s.status
		FROM semesters s
		JOIN course_offerings co ON co.semester_id = s.id
		WHERE co.id = $1`

	var status string
	if err := r.db.GetContext(ctx, &status, query, offeringID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrOfferingNotFound
		}
		return "", err
	}
	return status, nil
}

func (r *Repository) GetPassThreshold(ctx context.Context, offeringID uuid.UUID) (int, error) {
	query := `
		SELECT s.pass_threshold
		FROM semesters s
		JOIN course_offerings co ON co.semester_id = s.id
		WHERE co.id = $1`

	var threshold int
	if err := r.db.GetContext(ctx, &threshold, query, offeringID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrOfferingNotFound
		}
		return 0, err
	}
	return threshold, nil
}

func (r *Repository) IsOfferingFinalized(ctx context.Context, offeringID uuid.UUID) (bool, error) {
	query := `
		SELECT NOT EXISTS(
			SELECT 1 FROM course_enrollments
			WHERE offering_id = $1
				AND status = 'enrolled'
		)`

	var finalized bool
	err := r.db.GetContext(ctx, &finalized, query, offeringID)
	return finalized, err
}

func (r *Repository) GetStudentExamScores(ctx context.Context, studentID uuid.UUID, examIDs []uuid.UUID) (map[uuid.UUID]ExamScore, error) {
	if len(examIDs) == 0 {
		return make(map[uuid.UUID]ExamScore), nil
	}

	query := `
		SELECT
			e.id AS exam_id,
			ea.total_score,
			e.total_score AS max_score
		FROM exams e
		LEFT JOIN students st ON st.user_id = $1
		LEFT JOIN exam_attempts ea ON ea.exam_id = e.id AND ea.student_id = st.id AND ea.submitted_at IS NOT NULL
		WHERE e.id = ANY($2)`

	type row struct {
		ExamID     uuid.UUID `db:"exam_id"`
		TotalScore *float64  `db:"total_score"`
		MaxScore   float64   `db:"max_score"`
	}

	var rows []row
	if err := r.db.SelectContext(ctx, &rows, query, studentID, examIDs); err != nil {
		return nil, err
	}

	scores := make(map[uuid.UUID]ExamScore)
	for _, r := range rows {
		scores[r.ExamID] = ExamScore(r)
	}
	return scores, nil
}

func (r *Repository) ExamsBelongToOffering(ctx context.Context, offeringID uuid.UUID, examIDs []uuid.UUID) (bool, error) {
	if len(examIDs) == 0 {
		return true, nil
	}

	query := `
		SELECT COUNT(*) = $1
		FROM exams
		WHERE id = ANY($2) AND offering_id = $3`

	var valid bool
	err := r.db.GetContext(ctx, &valid, query, len(examIDs), examIDs, offeringID)
	return valid, err
}

func (r *Repository) GetStudentAssignmentAverage(ctx context.Context, studentID, offeringID uuid.UUID) (float64, bool, error) {
	query := `
		SELECT
			COALESCE(SUM(ass.score), 0) AS total_score,
			COALESCE(SUM(a.max_score), 0) AS max_possible,
			COUNT(a.id) AS assignment_count
		FROM assignments a
		LEFT JOIN assignment_submissions ass ON ass.assignment_id = a.id AND ass.student_id = $1
		WHERE a.offering_id = $2`

	var result struct {
		TotalScore      float64 `db:"total_score"`
		MaxPossible     float64 `db:"max_possible"`
		AssignmentCount int     `db:"assignment_count"`
	}

	if err := r.db.GetContext(ctx, &result, query, studentID, offeringID); err != nil {
		return 0, false, err
	}

	if result.AssignmentCount == 0 {
		return 0, false, nil
	}

	if result.MaxPossible == 0 {
		return 0, true, nil
	}

	percentage := (result.TotalScore / result.MaxPossible) * 100
	return percentage, true, nil
}

func (r *Repository) GetStudentAttendanceRate(ctx context.Context, studentID, offeringID uuid.UUID) (float64, error) {
	query := `
		SELECT
			COALESCE(SUM(l.duration_hours), 0) AS total_hours,
			COALESCE(SUM(
				CASE
					WHEN er.status = 'approved' THEN 0
					ELSE l.duration_hours * a.percentage / 100.0
				END
			), 0) AS attended_hours,
			COALESCE(SUM(
				CASE WHEN er.status = 'approved' THEN l.duration_hours ELSE 0 END
			), 0) AS excused_hours
		FROM lessons l
		LEFT JOIN attendance a ON a.lesson_id = l.id AND a.student_id = $1
		LEFT JOIN excuse_requests er ON er.lesson_id = l.id AND er.student_id = $1
		WHERE l.offering_id = $2 AND l.duration_hours IS NOT NULL`

	var result struct {
		TotalHours    float64 `db:"total_hours"`
		AttendedHours float64 `db:"attended_hours"`
		ExcusedHours  float64 `db:"excused_hours"`
	}

	if err := r.db.GetContext(ctx, &result, query, studentID, offeringID); err != nil {
		return 0, err
	}

	effectiveTotal := result.TotalHours - result.ExcusedHours
	if effectiveTotal <= 0 {
		return 100, nil
	}

	return (result.AttendedHours / effectiveTotal) * 100, nil
}

func (r *Repository) GetEnrolledStudentIDs(ctx context.Context, offeringID uuid.UUID) ([]uuid.UUID, error) {
	query := `
		SELECT student_id
		FROM course_enrollments
		WHERE offering_id = $1 AND status = 'enrolled'`

	var ids []uuid.UUID
	if err := r.db.SelectContext(ctx, &ids, query, offeringID); err != nil {
		return nil, err
	}
	return ids, nil
}

func (r *Repository) HasUngradedAttempts(ctx context.Context, offeringID uuid.UUID, examIDs []uuid.UUID) (bool, error) {
	if len(examIDs) == 0 {
		return false, nil
	}

	query := `
		SELECT EXISTS(
			SELECT 1 FROM exam_attempts ea
			JOIN exams e ON e.id = ea.exam_id
			WHERE e.offering_id = $1
				AND e.id = ANY($2)
				AND ea.submitted_at IS NOT NULL
				AND ea.total_score IS NULL
		)`

	var hasUngraded bool
	err := r.db.GetContext(ctx, &hasUngraded, query, offeringID, examIDs)
	return hasUngraded, err
}

func (r *Repository) HasUngradedSubmissions(ctx context.Context, offeringID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM assignment_submissions ass
			JOIN assignments a ON a.id = ass.assignment_id
			WHERE a.offering_id = $1
				AND ass.submitted_at IS NOT NULL
				AND ass.score IS NULL
		)`

	var hasUngraded bool
	err := r.db.GetContext(ctx, &hasUngraded, query, offeringID)
	return hasUngraded, err
}
