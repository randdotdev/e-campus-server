package qa

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/ranjdotdev/e-campus-server/internal/pagination"
)

type QuestionRepo struct {
	db *sqlx.DB
}

func NewQuestionRepository(db *sqlx.DB) *QuestionRepo {
	return &QuestionRepo{db: db}
}

func (r *QuestionRepo) Create(ctx context.Context, q *Question) error {
	query := `
		INSERT INTO qa_questions (id, offering_id, title, body, is_anonymous, is_faq, status, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := r.db.ExecContext(ctx, query,
		q.ID, q.OfferingID, q.Title, q.Body, q.IsAnonymous, q.IsFAQ, q.Status, q.CreatedBy, q.CreatedAt)
	return err
}

func (r *QuestionRepo) GetByID(ctx context.Context, id uuid.UUID) (*Question, error) {
	var q Question
	query := `SELECT * FROM qa_questions WHERE id = $1`

	if err := r.db.GetContext(ctx, &q, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &q, nil
}

func (r *QuestionRepo) GetByIDWithAuthor(ctx context.Context, id uuid.UUID) (*QuestionWithAuthor, error) {
	var q QuestionWithAuthor
	query := `
		SELECT q.*,
			u.full_name_en AS author_name,
			u.full_name_local AS author_name_local
		FROM qa_questions q
		JOIN users u ON q.created_by = u.id
		WHERE q.id = $1`

	if err := r.db.GetContext(ctx, &q, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &q, nil
}

func (r *QuestionRepo) Update(ctx context.Context, q *Question) error {
	query := `
		UPDATE qa_questions
		SET title = $2, body = $3, status = $4, updated_at = $5, edited_by = $6
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query,
		q.ID, q.Title, q.Body, q.Status, q.UpdatedAt, q.EditedBy)
	return err
}

func (r *QuestionRepo) SoftDelete(ctx context.Context, id uuid.UUID, deletedAt time.Time) error {
	query := `UPDATE qa_questions SET deleted_at = $2 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, deletedAt)
	return err
}

func (r *QuestionRepo) ListByOffering(ctx context.Context, offeringID uuid.UUID, status string, isFAQ *bool, params pagination.PageParams) ([]QuestionWithAuthor, bool, error) {
	var args []any
	argIndex := 1

	query := `
		SELECT q.*,
			u.full_name_en AS author_name,
			u.full_name_local AS author_name_local
		FROM qa_questions q
		JOIN users u ON q.created_by = u.id
		WHERE q.offering_id = $1 AND q.status = $2 AND q.deleted_at IS NULL`
	args = append(args, offeringID, status)
	argIndex += 2

	if isFAQ != nil {
		query += fmt.Sprintf(" AND q.is_faq = $%d", argIndex)
		args = append(args, *isFAQ)
		argIndex++
	}

	if params.Cursor != "" {
		cursorTime, cursorID, err := pagination.DecodeCursor(params.Cursor)
		if err != nil {
			return nil, false, err
		}
		query += fmt.Sprintf(" AND (q.created_at, q.id) < ($%d, $%d)", argIndex, argIndex+1)
		args = append(args, cursorTime, cursorID)
		argIndex += 2
	}

	query += " ORDER BY q.created_at DESC, q.id DESC"
	query += fmt.Sprintf(" LIMIT $%d", argIndex)
	args = append(args, params.Limit+1)

	var questions []QuestionWithAuthor
	if err := r.db.SelectContext(ctx, &questions, query, args...); err != nil {
		return nil, false, err
	}

	hasMore := len(questions) > params.Limit
	if hasMore {
		questions = questions[:params.Limit]
	}

	return questions, hasMore, nil
}

func (r *QuestionRepo) ListPending(ctx context.Context, offeringID uuid.UUID, params pagination.PageParams) ([]QuestionWithAuthor, bool, error) {
	var args []any
	argIndex := 1

	query := `
		SELECT q.*,
			u.full_name_en AS author_name,
			u.full_name_local AS author_name_local
		FROM qa_questions q
		JOIN users u ON q.created_by = u.id
		WHERE q.offering_id = $1 AND q.status = $2 AND q.deleted_at IS NULL`
	args = append(args, offeringID, StatusPending)
	argIndex += 2

	if params.Cursor != "" {
		cursorTime, cursorID, err := pagination.DecodeCursor(params.Cursor)
		if err != nil {
			return nil, false, err
		}
		query += fmt.Sprintf(" AND (q.created_at, q.id) < ($%d, $%d)", argIndex, argIndex+1)
		args = append(args, cursorTime, cursorID)
		argIndex += 2
	}

	query += " ORDER BY q.created_at DESC, q.id DESC"
	query += fmt.Sprintf(" LIMIT $%d", argIndex)
	args = append(args, params.Limit+1)

	var questions []QuestionWithAuthor
	if err := r.db.SelectContext(ctx, &questions, query, args...); err != nil {
		return nil, false, err
	}

	hasMore := len(questions) > params.Limit
	if hasMore {
		questions = questions[:params.Limit]
	}

	return questions, hasMore, nil
}

type AnswerRepo struct {
	db *sqlx.DB
}

func NewAnswerRepository(db *sqlx.DB) *AnswerRepo {
	return &AnswerRepo{db: db}
}

func (r *AnswerRepo) Create(ctx context.Context, a *Answer) error {
	query := `
		INSERT INTO qa_answers (id, question_id, body, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5)`

	_, err := r.db.ExecContext(ctx, query, a.ID, a.QuestionID, a.Body, a.CreatedBy, a.CreatedAt)
	return err
}

func (r *AnswerRepo) GetByQuestionID(ctx context.Context, questionID uuid.UUID) (*Answer, error) {
	var a Answer
	query := `SELECT * FROM qa_answers WHERE question_id = $1`

	if err := r.db.GetContext(ctx, &a, query, questionID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

func (r *AnswerRepo) GetByQuestionIDWithAuthor(ctx context.Context, questionID uuid.UUID) (*AnswerWithAuthor, error) {
	var a AnswerWithAuthor
	query := `
		SELECT a.*,
			u.full_name_en AS author_name,
			u.full_name_local AS author_name_local
		FROM qa_answers a
		JOIN users u ON a.created_by = u.id
		WHERE a.question_id = $1`

	if err := r.db.GetContext(ctx, &a, query, questionID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

func (r *AnswerRepo) Update(ctx context.Context, a *Answer) error {
	query := `
		UPDATE qa_answers
		SET body = $2, updated_by = $3, updated_at = $4
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, a.ID, a.Body, a.UpdatedBy, a.UpdatedAt)
	return err
}

type QuestionAttachmentRepo struct {
	db *sqlx.DB
}

func NewQuestionAttachmentRepository(db *sqlx.DB) *QuestionAttachmentRepo {
	return &QuestionAttachmentRepo{db: db}
}

func (r *QuestionAttachmentRepo) CreateBatch(ctx context.Context, questionID uuid.UUID, attachments []QuestionAttachment) error {
	if len(attachments) == 0 {
		return nil
	}

	query := `INSERT INTO qa_question_attachments (id, question_id, file_path, file_name, file_size, mime_type, created_at) VALUES `
	args := make([]any, 0, len(attachments)*7)

	for i, a := range attachments {
		if i > 0 {
			query += ", "
		}
		base := i * 7
		query += fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d)", base+1, base+2, base+3, base+4, base+5, base+6, base+7)
		args = append(args, a.ID, questionID, a.FilePath, a.FileName, a.FileSize, a.MimeType, a.CreatedAt)
	}

	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *QuestionAttachmentRepo) ListByQuestionID(ctx context.Context, questionID uuid.UUID) ([]QuestionAttachment, error) {
	var attachments []QuestionAttachment
	query := `SELECT * FROM qa_question_attachments WHERE question_id = $1 ORDER BY created_at`

	if err := r.db.SelectContext(ctx, &attachments, query, questionID); err != nil {
		return nil, err
	}
	return attachments, nil
}

func (r *QuestionAttachmentRepo) ListByQuestionIDs(ctx context.Context, questionIDs []uuid.UUID) (map[uuid.UUID][]QuestionAttachment, error) {
	if len(questionIDs) == 0 {
		return make(map[uuid.UUID][]QuestionAttachment), nil
	}

	query, args, err := sqlx.In(`SELECT * FROM qa_question_attachments WHERE question_id IN (?) ORDER BY created_at`, questionIDs)
	if err != nil {
		return nil, err
	}
	query = r.db.Rebind(query)

	var attachments []QuestionAttachment
	if err := r.db.SelectContext(ctx, &attachments, query, args...); err != nil {
		return nil, err
	}

	result := make(map[uuid.UUID][]QuestionAttachment)
	for _, a := range attachments {
		result[a.QuestionID] = append(result[a.QuestionID], a)
	}
	return result, nil
}

func (r *QuestionAttachmentRepo) DeleteByQuestionID(ctx context.Context, questionID uuid.UUID) error {
	query := `DELETE FROM qa_question_attachments WHERE question_id = $1`
	_, err := r.db.ExecContext(ctx, query, questionID)
	return err
}

type AnswerAttachmentRepo struct {
	db *sqlx.DB
}

func NewAnswerAttachmentRepository(db *sqlx.DB) *AnswerAttachmentRepo {
	return &AnswerAttachmentRepo{db: db}
}

func (r *AnswerAttachmentRepo) CreateBatch(ctx context.Context, answerID uuid.UUID, attachments []AnswerAttachment) error {
	if len(attachments) == 0 {
		return nil
	}

	query := `INSERT INTO qa_answer_attachments (id, answer_id, file_path, file_name, file_size, mime_type, created_at) VALUES `
	args := make([]any, 0, len(attachments)*7)

	for i, a := range attachments {
		if i > 0 {
			query += ", "
		}
		base := i * 7
		query += fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d)", base+1, base+2, base+3, base+4, base+5, base+6, base+7)
		args = append(args, a.ID, answerID, a.FilePath, a.FileName, a.FileSize, a.MimeType, a.CreatedAt)
	}

	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *AnswerAttachmentRepo) ListByAnswerID(ctx context.Context, answerID uuid.UUID) ([]AnswerAttachment, error) {
	var attachments []AnswerAttachment
	query := `SELECT * FROM qa_answer_attachments WHERE answer_id = $1 ORDER BY created_at`

	if err := r.db.SelectContext(ctx, &attachments, query, answerID); err != nil {
		return nil, err
	}
	return attachments, nil
}

func (r *AnswerAttachmentRepo) DeleteByAnswerID(ctx context.Context, answerID uuid.UUID) error {
	query := `DELETE FROM qa_answer_attachments WHERE answer_id = $1`
	_, err := r.db.ExecContext(ctx, query, answerID)
	return err
}

type RejectionRepo struct {
	db *sqlx.DB
}

func NewRejectionRepository(db *sqlx.DB) *RejectionRepo {
	return &RejectionRepo{db: db}
}

func (r *RejectionRepo) Create(ctx context.Context, qr *QuestionRejection) error {
	query := `
		INSERT INTO qa_rejections (question_id, reason, rejected_by, rejected_at)
		VALUES ($1, $2, $3, $4)`

	_, err := r.db.ExecContext(ctx, query, qr.QuestionID, qr.Reason, qr.RejectedBy, qr.RejectedAt)
	return err
}

func (r *RejectionRepo) GetByQuestionID(ctx context.Context, questionID uuid.UUID) (*QuestionRejection, error) {
	var qr QuestionRejection
	query := `SELECT * FROM qa_rejections WHERE question_id = $1`

	if err := r.db.GetContext(ctx, &qr, query, questionID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &qr, nil
}

func (r *RejectionRepo) GetByQuestionIDWithUser(ctx context.Context, questionID uuid.UUID) (*QuestionRejectionWithUser, error) {
	var qr QuestionRejectionWithUser
	query := `
		SELECT qr.*,
			u.full_name_en AS rejected_by_name,
			u.full_name_local AS rejected_by_name_local
		FROM qa_rejections qr
		JOIN users u ON qr.rejected_by = u.id
		WHERE qr.question_id = $1`

	if err := r.db.GetContext(ctx, &qr, query, questionID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &qr, nil
}

type OfferingRepo struct {
	db *sqlx.DB
}

func NewOfferingRepository(db *sqlx.DB) *OfferingRepo {
	return &OfferingRepo{db: db}
}

func (r *OfferingRepo) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM course_offerings WHERE id = $1)`
	err := r.db.GetContext(ctx, &exists, query, id)
	return exists, err
}
