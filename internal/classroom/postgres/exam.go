package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/randdotdev/e-campus-server/internal/classroom"
)

// ExamRepository is the SQL adapter for exams and attempts.
//
// The one-open-attempt rule is a partial unique index on
// (exam_id, student_id) WHERE submitted_at IS NULL; the attempt cap is a
// counting subquery inside the insert. Both races settle in the database.
type ExamRepository struct {
	db *sqlx.DB
}

func NewExamRepository(db *sqlx.DB) *ExamRepository {
	return &ExamRepository{db: db}
}

var (
	_ classroom.ExamRepository  = (*ExamRepository)(nil)
	_ classroom.ExamScoreReader = (*ExamRepository)(nil)
)

func (r *ExamRepository) CreateExam(ctx context.Context, e *classroom.Exam) error {
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO exams (id, offering_id, section_id, title, description, type, mode,
			questions, total_score, duration_minutes, shuffle_questions, shuffle_options,
			show_results, max_attempts, available_from, available_until, status,
			created_by, created_at)
		VALUES (:id, :offering_id, :section_id, :title, :description, :type, :mode,
			:questions, :total_score, :duration_minutes, :shuffle_questions, :shuffle_options,
			:show_results, :max_attempts, :available_from, :available_until, :status,
			:created_by, :created_at)`, e)
	return err
}

func (r *ExamRepository) GetExam(ctx context.Context, offeringID, id uuid.UUID) (*classroom.Exam, error) {
	var e classroom.Exam
	err := r.db.GetContext(ctx, &e,
		`SELECT * FROM exams WHERE id = $1 AND offering_id = $2`, id, offeringID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, classroom.ErrExamNotFound
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *ExamRepository) ListExams(ctx context.Context, offeringID uuid.UUID, publishedOnly bool) ([]classroom.Exam, error) {
	exams := []classroom.Exam{}
	query := `SELECT * FROM exams WHERE offering_id = $1`
	if publishedOnly {
		query += ` AND status <> 'draft'`
	}
	query += ` ORDER BY created_at DESC`
	err := r.db.SelectContext(ctx, &exams, query, offeringID)
	return exams, err
}

// UpdateExam is the version CAS with the draft guard folded in: a version
// miss on an existing draft is ErrConflict; a non-draft row never matches.
func (r *ExamRepository) UpdateExam(ctx context.Context, e *classroom.Exam, expectedVersion int64) (int64, error) {
	return scanVersion(r.db.QueryRowxContext(ctx, `
		UPDATE exams SET
			section_id = $1, title = $2, description = $3, questions = $4,
			total_score = $5, duration_minutes = $6, shuffle_questions = $7,
			shuffle_options = $8, show_results = $9, max_attempts = $10,
			available_from = $11, available_until = $12, version = version + 1
		WHERE id = $13 AND version = $14 AND status = 'draft'
		RETURNING version`,
		e.SectionID, e.Title, e.Description, e.Questions,
		e.TotalScore, e.DurationMinutes, e.ShuffleQuestions,
		e.ShuffleOptions, e.ShowResults, e.MaxAttempts,
		e.AvailableFrom, e.AvailableUntil,
		e.ID, expectedVersion))
}

func (r *ExamRepository) DeleteExam(ctx context.Context, offeringID, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM exams WHERE id = $1 AND offering_id = $2 AND status = 'draft'`, id, offeringID)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		if _, gerr := r.GetExam(ctx, offeringID, id); gerr != nil {
			return gerr
		}
		return classroom.ErrExamNotDraft
	}
	return nil
}

func (r *ExamRepository) PublishExam(ctx context.Context, offeringID, id uuid.UUID, at time.Time) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE exams SET status = 'published', published_at = $1
		WHERE id = $2 AND offering_id = $3 AND status = 'draft'`, at, id, offeringID)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		if _, gerr := r.GetExam(ctx, offeringID, id); gerr != nil {
			return gerr
		}
		return classroom.ErrExamNotDraft
	}
	return nil
}

func (r *ExamRepository) CloseExam(ctx context.Context, offeringID, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE exams SET status = 'closed'
		WHERE id = $1 AND offering_id = $2 AND status = 'published'`, id, offeringID)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		if _, gerr := r.GetExam(ctx, offeringID, id); gerr != nil {
			return gerr
		}
		return classroom.ErrExamNotPublished
	}
	return nil
}

// StartAttempt inserts only while the submitted count is under the cap;
// the partial unique index turns a concurrent double-start into a unique
// violation, answered with the already-open attempt.
func (r *ExamRepository) StartAttempt(ctx context.Context, examID, studentID uuid.UUID, maxAttempts int, at time.Time) (*classroom.Attempt, error) {
	if open, err := r.openAttempt(ctx, examID, studentID); err == nil && open != nil {
		return open, nil
	} else if err != nil {
		return nil, err
	}

	var a classroom.Attempt
	err := r.db.QueryRowxContext(ctx, `
		INSERT INTO exam_attempts (exam_id, student_id, started_at)
		SELECT $1, $2, $3
		WHERE (SELECT COUNT(*) FROM exam_attempts
		       WHERE exam_id = $1 AND student_id = $2 AND submitted_at IS NOT NULL) < $4
		RETURNING *`, examID, studentID, at, maxAttempts,
	).StructScan(&a)
	if isUniqueViolation(err) {
		if open, oerr := r.openAttempt(ctx, examID, studentID); oerr == nil && open != nil {
			return open, nil
		}
		return nil, classroom.ErrConflict
	}
	if errors.Is(err, sql.ErrNoRows) {
		return nil, classroom.ErrMaxAttemptsReached
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *ExamRepository) openAttempt(ctx context.Context, examID, studentID uuid.UUID) (*classroom.Attempt, error) {
	var a classroom.Attempt
	err := r.db.GetContext(ctx, &a, `
		SELECT * FROM exam_attempts
		WHERE exam_id = $1 AND student_id = $2 AND submitted_at IS NULL`, examID, studentID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *ExamRepository) GetAttempt(ctx context.Context, id uuid.UUID) (*classroom.Attempt, error) {
	var a classroom.Attempt
	err := r.db.GetContext(ctx, &a, `SELECT * FROM exam_attempts WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, classroom.ErrAttemptNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// GetStudentAttempt returns the student's latest attempt on the exam.
func (r *ExamRepository) GetStudentAttempt(ctx context.Context, examID, studentID uuid.UUID) (*classroom.Attempt, error) {
	var a classroom.Attempt
	err := r.db.GetContext(ctx, &a, `
		SELECT * FROM exam_attempts
		WHERE exam_id = $1 AND student_id = $2
		ORDER BY started_at DESC NULLS LAST LIMIT 1`, examID, studentID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, classroom.ErrAttemptNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *ExamRepository) ListAttempts(ctx context.Context, examID uuid.UUID) ([]classroom.AttemptWithStudent, error) {
	attempts := []classroom.AttemptWithStudent{}
	err := r.db.SelectContext(ctx, &attempts, `
		SELECT ea.*, u.full_name_en AS student_name, u.username AS student_username
		FROM exam_attempts ea
		JOIN users u ON u.id = ea.student_id
		WHERE ea.exam_id = $1
		ORDER BY ea.started_at DESC NULLS LAST`, examID)
	return attempts, err
}

func (r *ExamRepository) SaveAnswers(ctx context.Context, id uuid.UUID, answers json.RawMessage) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE exam_attempts SET answers = $1, updated_at = NOW()
		WHERE id = $2 AND submitted_at IS NULL`, answers, id)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return classroom.ErrAttemptSubmitted
	}
	return nil
}

func (r *ExamRepository) SubmitAttempt(ctx context.Context, id uuid.UUID, answers, scores json.RawMessage, totalScore *float64, at time.Time) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE exam_attempts
		SET answers = $1, scores = $2, total_score = $3, submitted_at = $4, updated_at = $4
		WHERE id = $5 AND submitted_at IS NULL`,
		answers, scores, totalScore, at, id)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return classroom.ErrAttemptSubmitted
	}
	return nil
}

func (r *ExamRepository) GradeAttempt(ctx context.Context, id uuid.UUID, scores json.RawMessage, totalScore float64, gradedBy uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE exam_attempts
		SET scores = $1, total_score = $2, graded_by = $3, graded_at = NOW(), updated_at = NOW()
		WHERE id = $4 AND submitted_at IS NOT NULL`,
		scores, totalScore, gradedBy, id)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return classroom.ErrAttemptNotFound
	}
	return nil
}

func (r *ExamRepository) UpdateAttemptReview(ctx context.Context, id uuid.UUID, visibility *classroom.ResultVisibility, visibleAt *time.Time, lateAccepted *bool) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE exam_attempts SET
			visibility = COALESCE($1, visibility),
			visible_at = COALESCE($2, visible_at),
			late_accepted = COALESCE($3, late_accepted)
		WHERE id = $4`, visibility, visibleAt, lateAccepted, id)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return classroom.ErrAttemptNotFound
	}
	return nil
}

// RecordResults stamps the exam used and upserts one submitted, graded
// attempt per student, all in one transaction.
func (r *ExamRepository) RecordResults(ctx context.Context, exam *classroom.Exam, results []classroom.ManualResult, visibility classroom.ResultVisibility, visibleAt *time.Time) error {
	return inTx(ctx, r.db, func(tx *sqlx.Tx) error {
		if _, err := tx.ExecContext(ctx,
			`UPDATE exams SET used_at = NOW() WHERE id = $1 AND used_at IS NULL`, exam.ID); err != nil {
			return err
		}
		for _, res := range results {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO exam_attempts (exam_id, student_id, total_score, started_at,
					submitted_at, visibility, visible_at)
				VALUES ($1, $2, $3, NOW(), NOW(), $4, $5)
				ON CONFLICT (exam_id, student_id) WHERE submitted_at IS NULL
				DO UPDATE SET total_score = EXCLUDED.total_score, submitted_at = NOW(),
					visibility = EXCLUDED.visibility, visible_at = EXCLUDED.visible_at`,
				exam.ID, res.StudentID, res.TotalScore, visibility, visibleAt); err != nil {
				if isForeignKeyViolation(err) {
					return fmt.Errorf("classroom: manual result for unknown student %s: %w", res.StudentID, classroom.ErrInvalidInput)
				}
				return err
			}
		}
		return nil
	})
}

// ── Grading reads (classroom.ExamScoreReader) ───────────────────────────────

// StudentExamScores reads best submitted results; exams without a
// submitted attempt come back with a nil score.
func (r *ExamRepository) StudentExamScores(ctx context.Context, userID uuid.UUID, examIDs []uuid.UUID) (map[uuid.UUID]classroom.ExamScore, error) {
	if len(examIDs) == 0 {
		return map[uuid.UUID]classroom.ExamScore{}, nil
	}
	query, args, err := sqlx.In(`
		SELECT e.id AS exam_id, e.total_score AS max_score, ea.total_score
		FROM exams e
		LEFT JOIN exam_attempts ea
			ON ea.exam_id = e.id AND ea.student_id = ? AND ea.submitted_at IS NOT NULL
		WHERE e.id IN (?)`, userID, examIDs)
	if err != nil {
		return nil, fmt.Errorf("classroom: build exam score lookup: %w", err)
	}
	var rows []struct {
		ExamID     uuid.UUID `db:"exam_id"`
		MaxScore   float64   `db:"max_score"`
		TotalScore *float64  `db:"total_score"`
	}
	if err := r.db.SelectContext(ctx, &rows, r.db.Rebind(query), args...); err != nil {
		return nil, err
	}
	// Several attempts may exist; keep the best.
	scores := make(map[uuid.UUID]classroom.ExamScore, len(rows))
	for _, row := range rows {
		current, ok := scores[row.ExamID]
		if !ok || better(row.TotalScore, current.TotalScore) {
			scores[row.ExamID] = classroom.ExamScore{TotalScore: row.TotalScore, MaxScore: row.MaxScore}
		}
	}
	return scores, nil
}

func better(a, b *float64) bool {
	if a == nil {
		return false
	}
	return b == nil || *a > *b
}

func (r *ExamRepository) ExamsBelongToOffering(ctx context.Context, offeringID uuid.UUID, examIDs []uuid.UUID) (bool, error) {
	if len(examIDs) == 0 {
		return true, nil
	}
	query, args, err := sqlx.In(
		`SELECT COUNT(*) FROM exams WHERE offering_id = ? AND id IN (?)`, offeringID, examIDs)
	if err != nil {
		return false, fmt.Errorf("classroom: build exam ownership check: %w", err)
	}
	var count int
	if err := r.db.GetContext(ctx, &count, r.db.Rebind(query), args...); err != nil {
		return false, err
	}
	return count == len(examIDs), nil
}

func (r *ExamRepository) HasUngradedAttempts(ctx context.Context, examIDs []uuid.UUID) (bool, error) {
	if len(examIDs) == 0 {
		return false, nil
	}
	query, args, err := sqlx.In(`
		SELECT EXISTS(
			SELECT 1 FROM exam_attempts
			WHERE exam_id IN (?) AND submitted_at IS NOT NULL AND total_score IS NULL
		)`, examIDs)
	if err != nil {
		return false, fmt.Errorf("classroom: build ungraded check: %w", err)
	}
	var exists bool
	err = r.db.GetContext(ctx, &exists, r.db.Rebind(query), args...)
	return exists, err
}
