package exam

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/randdotdev/e-campus-server/internal/pagination"
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// Question operations

func (r *Repository) CreateQuestion(ctx context.Context, q *Question) error {
	query := `
		INSERT INTO questions (course_code, text, image_url, type, options, correct, default_score, difficulty, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, is_active, created_at, updated_at`

	return r.db.QueryRowxContext(ctx, query,
		q.CourseCode, q.Text, q.ImageURL, q.Type, q.Options, q.Correct, q.DefaultScore, q.Difficulty, q.CreatedBy,
	).Scan(&q.ID, &q.IsActive, &q.CreatedAt, &q.UpdatedAt)
}

func (r *Repository) GetQuestion(ctx context.Context, id uuid.UUID) (*Question, error) {
	var q Question
	query := `SELECT * FROM questions WHERE id = $1`

	if err := r.db.GetContext(ctx, &q, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrQuestionNotFound
		}
		return nil, err
	}
	return &q, nil
}

func (r *Repository) ListQuestions(ctx context.Context, params pagination.PageParams, filters QuestionFilters) ([]Question, bool, error) {
	var conditions []string
	var args []any
	argN := 1

	if params.Cursor != "" {
		createdAt, id, err := pagination.DecodeCursor(params.Cursor)
		if err != nil {
			return nil, false, err
		}
		conditions = append(conditions, fmt.Sprintf("(created_at, id) < ($%d, $%d)", argN, argN+1))
		args = append(args, createdAt, id)
		argN += 2
	}
	if params.Query != "" {
		conditions = append(conditions, fmt.Sprintf("text ILIKE $%d", argN))
		args = append(args, "%"+pagination.EscapeLike(params.Query)+"%")
		argN++
	}
	if filters.CourseCode != nil {
		conditions = append(conditions, fmt.Sprintf("course_code = $%d", argN))
		args = append(args, *filters.CourseCode)
		argN++
	}
	if filters.Type != nil {
		conditions = append(conditions, fmt.Sprintf("type = $%d", argN))
		args = append(args, *filters.Type)
		argN++
	}
	if filters.Difficulty != nil {
		conditions = append(conditions, fmt.Sprintf("difficulty = $%d", argN))
		args = append(args, *filters.Difficulty)
		argN++
	}
	if filters.IsActive != nil {
		conditions = append(conditions, fmt.Sprintf("is_active = $%d", argN))
		args = append(args, *filters.IsActive)
		argN++
	}
	if filters.CreatedBy != nil {
		conditions = append(conditions, fmt.Sprintf("created_by = $%d", argN))
		args = append(args, *filters.CreatedBy)
		argN++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}
	query := fmt.Sprintf("SELECT * FROM questions %s ORDER BY created_at DESC, id DESC LIMIT $%d", where, argN)
	args = append(args, params.Limit+1)

	var questions []Question
	if err := r.db.SelectContext(ctx, &questions, query, args...); err != nil {
		return nil, false, err
	}

	hasMore := len(questions) > params.Limit
	if hasMore {
		questions = questions[:params.Limit]
	}

	return questions, hasMore, nil
}

func (r *Repository) UpdateQuestion(ctx context.Context, q *Question) error {
	query := `
		UPDATE questions
		SET text = $2, image_url = $3, options = $4, correct = $5, default_score = $6, difficulty = $7
		WHERE id = $1
		RETURNING updated_at`

	err := r.db.QueryRowxContext(ctx, query,
		q.ID, q.Text, q.ImageURL, q.Options, q.Correct, q.DefaultScore, q.Difficulty,
	).Scan(&q.UpdatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return ErrQuestionNotFound
	}
	return err
}

func (r *Repository) SoftDeleteQuestion(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE questions SET is_active = false WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrQuestionNotFound
	}
	return nil
}

func (r *Repository) GetQuestionsByCourseCode(ctx context.Context, courseCode string) ([]Question, error) {
	var questions []Question
	query := `SELECT * FROM questions WHERE course_code = $1 AND is_active = true`

	if err := r.db.SelectContext(ctx, &questions, query, courseCode); err != nil {
		return nil, err
	}
	return questions, nil
}

func (r *Repository) GetQuestionsByIDs(ctx context.Context, ids []uuid.UUID) ([]Question, error) {
	if len(ids) == 0 {
		return []Question{}, nil
	}

	query, args, err := sqlx.In(`SELECT * FROM questions WHERE id IN (?)`, ids)
	if err != nil {
		return nil, err
	}
	query = r.db.Rebind(query)

	var questions []Question
	if err := r.db.SelectContext(ctx, &questions, query, args...); err != nil {
		return nil, err
	}
	return questions, nil
}

// Exam operations

func (r *Repository) CreateExam(ctx context.Context, e *Exam) error {
	query := `
		INSERT INTO exams (offering_id, section_id, title, description, type, mode, questions, total_score,
			duration_minutes, shuffle_questions, shuffle_options, show_results, max_attempts,
			available_from, available_until, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		RETURNING id, status, created_at`

	mode := e.Mode
	if mode == "" {
		mode = ExamModeOnline
	}

	showResults := e.ShowResults
	if showResults == "" {
		showResults = ShowResultsAfterSubmit
	}

	maxAttempts := e.MaxAttempts
	if maxAttempts == 0 {
		maxAttempts = 1
	}

	return r.db.QueryRowxContext(ctx, query,
		e.OfferingID, e.SectionID, e.Title, e.Description, e.Type, mode,
		e.Questions, e.TotalScore, e.DurationMinutes, e.ShuffleQuestions, e.ShuffleOptions,
		showResults, maxAttempts, e.AvailableFrom, e.AvailableUntil, e.CreatedBy,
	).Scan(&e.ID, &e.Status, &e.CreatedAt)
}

func (r *Repository) GetExam(ctx context.Context, id uuid.UUID) (*Exam, error) {
	var e Exam
	query := `SELECT * FROM exams WHERE id = $1`

	if err := r.db.GetContext(ctx, &e, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrExamNotFound
		}
		return nil, err
	}
	return &e, nil
}

func (r *Repository) ListExams(ctx context.Context, params pagination.PageParams, filters ExamFilters) ([]Exam, bool, error) {
	var conditions []string
	var args []any
	argN := 1

	if params.Cursor != "" {
		createdAt, id, err := pagination.DecodeCursor(params.Cursor)
		if err != nil {
			return nil, false, err
		}
		conditions = append(conditions, fmt.Sprintf("(created_at, id) < ($%d, $%d)", argN, argN+1))
		args = append(args, createdAt, id)
		argN += 2
	}
	if filters.OfferingID != nil {
		conditions = append(conditions, fmt.Sprintf("offering_id = $%d", argN))
		args = append(args, *filters.OfferingID)
		argN++
	}
	if filters.SectionID != nil {
		conditions = append(conditions, fmt.Sprintf("section_id = $%d", argN))
		args = append(args, *filters.SectionID)
		argN++
	}
	if filters.Type != nil {
		conditions = append(conditions, fmt.Sprintf("type = $%d", argN))
		args = append(args, *filters.Type)
		argN++
	}
	if filters.Mode != nil {
		conditions = append(conditions, fmt.Sprintf("mode = $%d", argN))
		args = append(args, *filters.Mode)
		argN++
	}
	if filters.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argN))
		args = append(args, *filters.Status)
		argN++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}
	query := fmt.Sprintf("SELECT * FROM exams %s ORDER BY created_at DESC, id DESC LIMIT $%d", where, argN)
	args = append(args, params.Limit+1)

	var exams []Exam
	if err := r.db.SelectContext(ctx, &exams, query, args...); err != nil {
		return nil, false, err
	}

	hasMore := len(exams) > params.Limit
	if hasMore {
		exams = exams[:params.Limit]
	}

	return exams, hasMore, nil
}

func (r *Repository) UpdateExam(ctx context.Context, e *Exam) error {
	query := `
		UPDATE exams
		SET section_id = $2, title = $3, description = $4, questions = $5, total_score = $6,
			duration_minutes = $7, shuffle_questions = $8, shuffle_options = $9, show_results = $10,
			max_attempts = $11, available_from = $12, available_until = $13
		WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query,
		e.ID, e.SectionID, e.Title, e.Description, e.Questions, e.TotalScore,
		e.DurationMinutes, e.ShuffleQuestions, e.ShuffleOptions, e.ShowResults,
		e.MaxAttempts, e.AvailableFrom, e.AvailableUntil,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrExamNotFound
	}
	return nil
}

func (r *Repository) DeleteExam(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM exams WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrExamNotFound
	}
	return nil
}

func (r *Repository) PublishExam(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE exams SET status = $2, published_at = $3 WHERE id = $1 AND status = 'draft'`
	result, err := r.db.ExecContext(ctx, query, id, ExamStatusPublished, time.Now())
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrExamNotFound
	}
	return nil
}

func (r *Repository) CloseExam(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE exams SET status = $2 WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id, ExamStatusClosed)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrExamNotFound
	}
	return nil
}

func (r *Repository) SetExamUsedAt(ctx context.Context, id uuid.UUID, usedAt time.Time) error {
	query := `UPDATE exams SET used_at = $2 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, usedAt)
	return err
}

// Attempt operations

func (r *Repository) CreateAttempt(ctx context.Context, a *Attempt) error {
	query := `
		INSERT INTO exam_attempts (exam_id, student_id, started_at)
		VALUES ($1, $2, $3)
		RETURNING id, visibility`

	return r.db.QueryRowxContext(ctx, query,
		a.ExamID, a.StudentID, a.StartedAt,
	).Scan(&a.ID, &a.Visibility)
}

func (r *Repository) GetAttempt(ctx context.Context, id uuid.UUID) (*Attempt, error) {
	var a Attempt
	query := `SELECT * FROM exam_attempts WHERE id = $1`

	if err := r.db.GetContext(ctx, &a, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAttemptNotFound
		}
		return nil, err
	}
	return &a, nil
}

func (r *Repository) GetAttemptByExamAndStudent(ctx context.Context, examID, studentID uuid.UUID) (*Attempt, error) {
	var a Attempt
	query := `SELECT * FROM exam_attempts WHERE exam_id = $1 AND student_id = $2`

	if err := r.db.GetContext(ctx, &a, query, examID, studentID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAttemptNotFound
		}
		return nil, err
	}
	return &a, nil
}

func (r *Repository) ListAttempts(ctx context.Context, params pagination.PageParams, filters AttemptFilters) ([]Attempt, bool, error) {
	joins := ""
	if filters.Query != "" {
		joins = "JOIN students s ON a.student_id = s.id JOIN users u ON s.user_id = u.id"
	}

	var conditions []string
	var args []any
	argN := 1

	if params.Cursor != "" {
		createdAt, id, err := pagination.DecodeCursor(params.Cursor)
		if err != nil {
			return nil, false, err
		}
		conditions = append(conditions, fmt.Sprintf("(a.started_at, a.id) < ($%d, $%d)", argN, argN+1))
		args = append(args, createdAt, id)
		argN += 2
	}
	if filters.Query != "" {
		conditions = append(conditions, fmt.Sprintf("(u.full_name_en ILIKE $%d OR u.full_name_local ILIKE $%d OR u.email ILIKE $%d)", argN, argN, argN))
		args = append(args, "%"+pagination.EscapeLike(filters.Query)+"%")
		argN++
	}
	if filters.ExamID != nil {
		conditions = append(conditions, fmt.Sprintf("a.exam_id = $%d", argN))
		args = append(args, *filters.ExamID)
		argN++
	}
	if filters.StudentID != nil {
		conditions = append(conditions, fmt.Sprintf("a.student_id = $%d", argN))
		args = append(args, *filters.StudentID)
		argN++
	}
	if filters.IsSubmitted != nil {
		if *filters.IsSubmitted {
			conditions = append(conditions, "a.submitted_at IS NOT NULL")
		} else {
			conditions = append(conditions, "a.submitted_at IS NULL")
		}
	}
	if filters.IsGraded != nil {
		if *filters.IsGraded {
			conditions = append(conditions, "a.graded_at IS NOT NULL")
		} else {
			conditions = append(conditions, "a.graded_at IS NULL")
		}
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}
	query := fmt.Sprintf(
		"SELECT a.* FROM exam_attempts a %s %s ORDER BY a.started_at DESC, a.id DESC LIMIT $%d",
		joins, where, argN,
	)
	args = append(args, params.Limit+1)

	var attempts []Attempt
	if err := r.db.SelectContext(ctx, &attempts, query, args...); err != nil {
		return nil, false, err
	}

	hasMore := len(attempts) > params.Limit
	if hasMore {
		attempts = attempts[:params.Limit]
	}

	return attempts, hasMore, nil
}

func (r *Repository) SaveAnswers(ctx context.Context, id uuid.UUID, answers json.RawMessage) error {
	query := `UPDATE exam_attempts SET answers = $2, updated_at = $3 WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id, answers, time.Now())
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrAttemptNotFound
	}
	return nil
}

func (r *Repository) SubmitAttempt(ctx context.Context, id uuid.UUID, answers, scores json.RawMessage, totalScore *float64) error {
	query := `
		UPDATE exam_attempts
		SET answers = $2, scores = $3, total_score = $4, submitted_at = $5, updated_at = $5
		WHERE id = $1`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query, id, answers, scores, totalScore, now)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrAttemptNotFound
	}
	return nil
}

func (r *Repository) SetLateDecision(ctx context.Context, id uuid.UUID, accepted bool) error {
	query := `UPDATE exam_attempts SET late_accepted = $2 WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id, accepted)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrAttemptNotFound
	}
	return nil
}

func (r *Repository) GradeAttempt(ctx context.Context, id uuid.UUID, scores json.RawMessage, totalScore float64, gradedBy uuid.UUID) error {
	query := `
		UPDATE exam_attempts
		SET scores = $2, total_score = $3, graded_by = $4, graded_at = $5
		WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id, scores, totalScore, gradedBy, time.Now())
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrAttemptNotFound
	}
	return nil
}

func (r *Repository) SetVisibility(ctx context.Context, id uuid.UUID, visibility string, visibleAt *time.Time) error {
	query := `UPDATE exam_attempts SET visibility = $2, visible_at = $3 WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id, visibility, visibleAt)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrAttemptNotFound
	}
	return nil
}

func (r *Repository) BulkCreateResults(ctx context.Context, examID uuid.UUID, results []BulkResultItem, visibility string, visibleAt *time.Time) error {
	if len(results) == 0 {
		return nil
	}

	now := time.Now()

	for _, item := range results {
		// Check if attempt exists for this student
		var existingID uuid.UUID
		checkQuery := `SELECT id FROM exam_attempts WHERE exam_id = $1 AND student_id = $2 LIMIT 1`
		err := r.db.GetContext(ctx, &existingID, checkQuery, examID, item.StudentID)

		if err == nil {
			// Update existing
			updateQuery := `UPDATE exam_attempts SET total_score = $2, submitted_at = $3, visibility = $4, visible_at = $5 WHERE id = $1`
			_, err = r.db.ExecContext(ctx, updateQuery, existingID, item.TotalScore, now, visibility, visibleAt)
		} else if errors.Is(err, sql.ErrNoRows) {
			// Insert new
			insertQuery := `INSERT INTO exam_attempts (exam_id, student_id, total_score, submitted_at, visibility, visible_at) VALUES ($1, $2, $3, $4, $5, $6)`
			_, err = r.db.ExecContext(ctx, insertQuery, examID, item.StudentID, item.TotalScore, now, visibility, visibleAt)
		}

		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) CountAttempts(ctx context.Context, examID, studentID uuid.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM exam_attempts WHERE exam_id = $1 AND student_id = $2`
	err := r.db.GetContext(ctx, &count, query, examID, studentID)
	return count, err
}

func (r *Repository) CountSubmittedAttempts(ctx context.Context, examID, studentID uuid.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM exam_attempts WHERE exam_id = $1 AND student_id = $2 AND submitted_at IS NOT NULL`
	err := r.db.GetContext(ctx, &count, query, examID, studentID)
	return count, err
}

func (r *Repository) GetInProgressAttempt(ctx context.Context, examID, studentID uuid.UUID) (*Attempt, error) {
	var a Attempt
	query := `SELECT * FROM exam_attempts WHERE exam_id = $1 AND student_id = $2 AND submitted_at IS NULL ORDER BY started_at DESC LIMIT 1`

	if err := r.db.GetContext(ctx, &a, query, examID, studentID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

// Helper operations

func (r *Repository) OfferingExists(ctx context.Context, offeringID uuid.UUID) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM course_offerings WHERE id = $1)`
	err := r.db.GetContext(ctx, &exists, query, offeringID)
	return exists, err
}

func (r *Repository) GetCourseCodeByOffering(ctx context.Context, offeringID uuid.UUID) (string, error) {
	var code string
	query := `
		SELECT c.code FROM courses c
		JOIN course_offerings o ON c.id = o.course_id
		WHERE o.id = $1`
	err := r.db.GetContext(ctx, &code, query, offeringID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrOfferingNotFound
	}
	return code, err
}

func (r *Repository) GetStudentByUserID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	var studentID uuid.UUID
	query := `SELECT id FROM students WHERE user_id = $1`
	err := r.db.GetContext(ctx, &studentID, query, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, ErrStudentNotFound
	}
	return studentID, err
}

func (r *Repository) GetUserIDByStudentID(ctx context.Context, studentID uuid.UUID) (uuid.UUID, error) {
	var userID uuid.UUID
	query := `SELECT user_id FROM students WHERE id = $1`
	err := r.db.GetContext(ctx, &userID, query, studentID)
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, ErrStudentNotFound
	}
	return userID, err
}

func (r *Repository) IsStudentEnrolled(ctx context.Context, offeringID, studentID uuid.UUID) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS(
			SELECT 1 FROM course_enrollments e
			JOIN students s ON e.student_id = s.user_id
			WHERE e.offering_id = $1 AND s.id = $2 AND e.status = 'enrolled'
		)`
	err := r.db.GetContext(ctx, &exists, query, offeringID, studentID)
	return exists, err
}

func (r *Repository) IsTeacher(ctx context.Context, offeringID, userID uuid.UUID) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM course_teachers WHERE offering_id = $1 AND user_id = $2)`
	err := r.db.GetContext(ctx, &exists, query, offeringID, userID)
	return exists, err
}
