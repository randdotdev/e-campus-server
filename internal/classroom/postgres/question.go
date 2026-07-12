package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/randdotdev/e-campus-server/internal/classroom"
	"github.com/randdotdev/e-campus-server/internal/shared/pagination"
)

// QuestionRepository is the SQL adapter for the exam question bank.
type QuestionRepository struct {
	db *sqlx.DB
}

func NewQuestionRepository(db *sqlx.DB) *QuestionRepository {
	return &QuestionRepository{db: db}
}

var _ classroom.QuestionRepository = (*QuestionRepository)(nil)

func (r *QuestionRepository) CreateQuestion(ctx context.Context, q *classroom.Question) error {
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO questions (id, course_code, text, image_id, type, options, correct,
			default_score, difficulty, is_active, created_by, created_at, updated_at)
		VALUES (:id, :course_code, :text, :image_id, :type, :options, :correct,
			:default_score, :difficulty, :is_active, :created_by, :created_at, :updated_at)`, q)
	return err
}

func (r *QuestionRepository) GetQuestion(ctx context.Context, courseCode string, id uuid.UUID) (*classroom.Question, error) {
	var q classroom.Question
	err := r.db.GetContext(ctx, &q,
		`SELECT * FROM questions WHERE id = $1 AND course_code = $2 AND is_active`, id, courseCode)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, classroom.ErrQuestionNotFound
	}
	if err != nil {
		return nil, err
	}
	return &q, nil
}

// ListQuestions filters in one static query; empty filter values disable
// their clause.
func (r *QuestionRepository) ListQuestions(ctx context.Context, courseCode string, filter classroom.QuestionFilter) ([]classroom.Question, error) {
	questions := []classroom.Question{}
	qType, difficulty := "", ""
	if filter.Type != nil {
		qType = string(*filter.Type)
	}
	if filter.Difficulty != nil {
		difficulty = string(*filter.Difficulty)
	}
	search := ""
	if filter.Search != "" {
		search = "%" + pagination.EscapeLike(filter.Search) + "%"
	}
	err := r.db.SelectContext(ctx, &questions, `
		SELECT * FROM questions
		WHERE course_code = $1 AND is_active
		  AND ($2 = '' OR type = $2)
		  AND ($3 = '' OR difficulty = $3)
		  AND ($4 = '' OR text ILIKE $4)
		ORDER BY created_at DESC`, courseCode, qType, difficulty, search)
	return questions, err
}

func (r *QuestionRepository) UpdateQuestion(ctx context.Context, q *classroom.Question) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE questions SET
			text = $1, image_id = $2, options = $3, correct = $4,
			default_score = $5, difficulty = $6, updated_at = NOW()
		WHERE id = $7 AND is_active`,
		q.Text, q.ImageID, q.Options, q.Correct, q.DefaultScore, q.Difficulty, q.ID)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return classroom.ErrQuestionNotFound
	}
	return nil
}

func (r *QuestionRepository) DeactivateQuestion(ctx context.Context, courseCode string, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE questions SET is_active = FALSE, updated_at = NOW()
		 WHERE id = $1 AND course_code = $2 AND is_active`, id, courseCode)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return classroom.ErrQuestionNotFound
	}
	return nil
}

// GetQuestionsByIDs resolves exam-embedded references; inactive questions
// still resolve — old exams keep working.
func (r *QuestionRepository) GetQuestionsByIDs(ctx context.Context, ids []uuid.UUID) ([]classroom.Question, error) {
	if len(ids) == 0 {
		return []classroom.Question{}, nil
	}
	query, args, err := sqlx.In(`SELECT * FROM questions WHERE id IN (?)`, ids)
	if err != nil {
		return nil, fmt.Errorf("classroom: build question lookup: %w", err)
	}
	questions := []classroom.Question{}
	err = r.db.SelectContext(ctx, &questions, r.db.Rebind(query), args...)
	return questions, err
}
