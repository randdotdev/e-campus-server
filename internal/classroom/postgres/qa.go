package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/randdotdev/e-campus-server/internal/classroom"
)

// QARepository is the SQL adapter for the question board. Question and
// answer attachments live in two tables behind one domain type; ParentID
// picks the table.
type QARepository struct {
	db *sqlx.DB
}

func NewQARepository(db *sqlx.DB) *QARepository {
	return &QARepository{db: db}
}

var _ classroom.QARepository = (*QARepository)(nil)

func insertQAAttachments(ctx context.Context, tx *sqlx.Tx, table, fkColumn string, atts []classroom.QAAttachment) error {
	for _, a := range atts {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO `+table+` (id, `+fkColumn+`, inode_id, display_name, order_index, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			a.ID, a.ParentID, a.InodeID, a.DisplayName, a.OrderIndex, a.CreatedAt); err != nil {
			return err
		}
	}
	return nil
}

func (r *QARepository) CreateQuestion(ctx context.Context, q *classroom.QAQuestion, attachments []classroom.QAAttachment) error {
	return inTx(ctx, r.db, func(tx *sqlx.Tx) error {
		if _, err := tx.NamedExecContext(ctx, `
			INSERT INTO qa_questions (id, offering_id, title, body, is_anonymous, is_faq,
				status, created_by, created_at)
			VALUES (:id, :offering_id, :title, :body, :is_anonymous, :is_faq,
				:status, :created_by, :created_at)`, q); err != nil {
			return err
		}
		return insertQAAttachments(ctx, tx, "qa_question_attachments", "question_id", attachments)
	})
}

func (r *QARepository) CreateFAQ(ctx context.Context, q *classroom.QAQuestion, a *classroom.QAAnswer, qAtts, aAtts []classroom.QAAttachment) error {
	return inTx(ctx, r.db, func(tx *sqlx.Tx) error {
		if _, err := tx.NamedExecContext(ctx, `
			INSERT INTO qa_questions (id, offering_id, title, body, is_anonymous, is_faq,
				status, created_by, created_at)
			VALUES (:id, :offering_id, :title, :body, :is_anonymous, :is_faq,
				:status, :created_by, :created_at)`, q); err != nil {
			return err
		}
		if _, err := tx.NamedExecContext(ctx, `
			INSERT INTO qa_answers (id, question_id, body, created_by, created_at)
			VALUES (:id, :question_id, :body, :created_by, :created_at)`, a); err != nil {
			return err
		}
		if err := insertQAAttachments(ctx, tx, "qa_question_attachments", "question_id", qAtts); err != nil {
			return err
		}
		return insertQAAttachments(ctx, tx, "qa_answer_attachments", "answer_id", aAtts)
	})
}

func (r *QARepository) GetQuestion(ctx context.Context, offeringID, id uuid.UUID) (*classroom.QAQuestion, error) {
	var q classroom.QAQuestion
	err := r.db.GetContext(ctx, &q, `
		SELECT * FROM qa_questions
		WHERE id = $1 AND offering_id = $2 AND deleted_at IS NULL`, id, offeringID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, classroom.ErrQuestionNotFound
	}
	if err != nil {
		return nil, err
	}
	return &q, nil
}

const qaQuestionViewQuery = `
	SELECT q.*, u.full_name_en AS author_name, u.username AS author_username,
	       u.avatar_url AS author_avatar
	FROM qa_questions q
	JOIN users u ON u.id = q.created_by`

func (r *QARepository) GetQuestionView(ctx context.Context, offeringID, id uuid.UUID) (*classroom.QAQuestionView, error) {
	var q classroom.QAQuestionView
	err := r.db.GetContext(ctx, &q,
		qaQuestionViewQuery+` WHERE q.id = $1 AND q.offering_id = $2 AND q.deleted_at IS NULL`,
		id, offeringID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, classroom.ErrQuestionNotFound
	}
	if err != nil {
		return nil, err
	}
	return &q, nil
}

func (r *QARepository) ListQuestions(ctx context.Context, offeringID uuid.UUID, filter classroom.QAFilter) ([]classroom.QAQuestionView, error) {
	questions := []classroom.QAQuestionView{}
	var mine uuid.UUID
	if filter.Mine != nil {
		mine = *filter.Mine
	}
	err := r.db.SelectContext(ctx, &questions, qaQuestionViewQuery+`
		WHERE q.offering_id = $1 AND q.deleted_at IS NULL
		  AND q.status = $2
		  AND ($3::boolean IS NULL OR q.is_faq = $3)
		  AND ($4::uuid = '00000000-0000-0000-0000-000000000000' OR q.created_by = $4)
		ORDER BY q.created_at DESC`,
		offeringID, filter.Status, filter.FAQ, mine)
	return questions, err
}

func (r *QARepository) UpdateQuestion(ctx context.Context, q *classroom.QAQuestion, expectedVersion int64) (int64, error) {
	return scanVersion(r.db.QueryRowxContext(ctx, `
		UPDATE qa_questions SET
			title = $1, body = $2, status = $3, updated_at = $4, edited_by = $5,
			version = version + 1
		WHERE id = $6 AND version = $7 AND deleted_at IS NULL
		RETURNING version`,
		q.Title, q.Body, q.Status, q.UpdatedAt, q.EditedBy,
		q.ID, expectedVersion))
}

func (r *QARepository) SoftDeleteQuestion(ctx context.Context, offeringID, id uuid.UUID, at time.Time) ([]uuid.UUID, error) {
	var inodeIDs []uuid.UUID
	err := inTx(ctx, r.db, func(tx *sqlx.Tx) error {
		if err := tx.SelectContext(ctx, &inodeIDs, `
			SELECT inode_id FROM qa_question_attachments WHERE question_id = $1
			UNION ALL
			SELECT qa.inode_id
			FROM qa_answer_attachments qa
			JOIN qa_answers a ON a.id = qa.answer_id
			WHERE a.question_id = $1`, id); err != nil {
			return err
		}
		// The attachment rows go too: a soft-deleted thread must not keep
		// its blobs pinned for a purge that may never come.
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM qa_question_attachments WHERE question_id = $1`, id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			DELETE FROM qa_answer_attachments qa
			USING qa_answers a
			WHERE qa.answer_id = a.id AND a.question_id = $1`, id); err != nil {
			return err
		}
		result, err := tx.ExecContext(ctx, `
			UPDATE qa_questions SET deleted_at = $1
			WHERE id = $2 AND offering_id = $3 AND deleted_at IS NULL`, at, id, offeringID)
		if err != nil {
			return err
		}
		if n, _ := result.RowsAffected(); n == 0 {
			return classroom.ErrQuestionNotFound
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return inodeIDs, nil
}

// AnswerQuestion upserts the single answer, flips the question, and
// replaces answer attachments — one transaction. The rejected guard rides
// the question UPDATE's WHERE.
func (r *QARepository) AnswerQuestion(ctx context.Context, questionID uuid.UUID, a *classroom.QAAnswer, attachments []classroom.QAAttachment, questionEdit *string, editorID uuid.UUID) ([]uuid.UUID, error) {
	var replaced []uuid.UUID
	err := inTx(ctx, r.db, func(tx *sqlx.Tx) error {
		result, err := tx.ExecContext(ctx, `
			UPDATE qa_questions SET
				status = 'answered', updated_at = NOW(),
				body = COALESCE($1, body),
				edited_by = CASE WHEN $1 IS NULL THEN edited_by ELSE $2 END
			WHERE id = $3 AND status <> 'rejected' AND deleted_at IS NULL`,
			questionEdit, editorID, questionID)
		if err != nil {
			return err
		}
		if n, _ := result.RowsAffected(); n == 0 {
			var status string
			if err := tx.GetContext(ctx, &status,
				`SELECT status FROM qa_questions WHERE id = $1 AND deleted_at IS NULL`, questionID); err != nil {
				return classroom.ErrQuestionNotFound
			}
			return classroom.ErrQuestionRejected
		}

		var answerID uuid.UUID
		if err := tx.QueryRowxContext(ctx, `
			INSERT INTO qa_answers (id, question_id, body, created_by, created_at)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (question_id) DO UPDATE
				SET body = EXCLUDED.body, updated_by = $4, updated_at = NOW()
			RETURNING id`,
			a.ID, questionID, a.Body, a.CreatedBy, a.CreatedAt,
		).Scan(&answerID); err != nil {
			return err
		}
		a.ID = answerID

		if err := tx.SelectContext(ctx, &replaced,
			`DELETE FROM qa_answer_attachments WHERE answer_id = $1 RETURNING inode_id`,
			answerID); err != nil {
			return err
		}
		for i := range attachments {
			attachments[i].ParentID = answerID
		}
		return insertQAAttachments(ctx, tx, "qa_answer_attachments", "answer_id", attachments)
	})
	if err != nil {
		return nil, err
	}
	return replaced, nil
}

func (r *QARepository) GetAnswerView(ctx context.Context, questionID uuid.UUID) (*classroom.QAAnswerView, error) {
	var a classroom.QAAnswerView
	err := r.db.GetContext(ctx, &a, `
		SELECT ans.*, u.full_name_en AS author_name, u.username AS author_username,
		       u.avatar_url AS author_avatar
		FROM qa_answers ans
		JOIN users u ON u.id = ans.created_by
		WHERE ans.question_id = $1`, questionID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, classroom.ErrAnswerNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// RejectQuestion flips pending → rejected and records why, atomically.
func (r *QARepository) RejectQuestion(ctx context.Context, rej *classroom.QARejection) error {
	return inTx(ctx, r.db, func(tx *sqlx.Tx) error {
		result, err := tx.ExecContext(ctx, `
			UPDATE qa_questions SET status = 'rejected', updated_at = NOW()
			WHERE id = $1 AND status = 'pending' AND deleted_at IS NULL`, rej.QuestionID)
		if err != nil {
			return err
		}
		if n, _ := result.RowsAffected(); n == 0 {
			return classroom.ErrQuestionNotPending
		}
		_, err = tx.ExecContext(ctx, `
			INSERT INTO qa_rejections (question_id, reason, rejected_by, rejected_at)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (question_id) DO UPDATE
				SET reason = EXCLUDED.reason, rejected_by = EXCLUDED.rejected_by,
				    rejected_at = EXCLUDED.rejected_at`,
			rej.QuestionID, rej.Reason, rej.RejectedBy, rej.RejectedAt)
		return err
	})
}

func (r *QARepository) GetRejection(ctx context.Context, questionID uuid.UUID) (*classroom.QARejection, error) {
	var rej classroom.QARejection
	err := r.db.GetContext(ctx, &rej,
		`SELECT * FROM qa_rejections WHERE question_id = $1`, questionID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, classroom.ErrQuestionNotFound
	}
	if err != nil {
		return nil, err
	}
	return &rej, nil
}

func (r *QARepository) ListQuestionAttachments(ctx context.Context, questionID uuid.UUID) ([]classroom.QAAttachment, error) {
	atts := []classroom.QAAttachment{}
	err := r.db.SelectContext(ctx, &atts, `
		SELECT id, question_id AS parent_id, inode_id, display_name, order_index, created_at
		FROM qa_question_attachments WHERE question_id = $1 ORDER BY order_index`, questionID)
	return atts, err
}

func (r *QARepository) ListAnswerAttachments(ctx context.Context, answerID uuid.UUID) ([]classroom.QAAttachment, error) {
	atts := []classroom.QAAttachment{}
	err := r.db.SelectContext(ctx, &atts, `
		SELECT id, answer_id AS parent_id, inode_id, display_name, order_index, created_at
		FROM qa_answer_attachments WHERE answer_id = $1 ORDER BY order_index`, answerID)
	return atts, err
}

// GetAttachment looks in both attachment tables; ParentID disambiguates.
func (r *QARepository) GetAttachment(ctx context.Context, parentID, id uuid.UUID) (*classroom.QAAttachment, error) {
	var a classroom.QAAttachment
	err := r.db.GetContext(ctx, &a, `
		SELECT id, question_id AS parent_id, inode_id, display_name, order_index, created_at
		FROM qa_question_attachments WHERE id = $1 AND question_id = $2
		UNION ALL
		SELECT id, answer_id AS parent_id, inode_id, display_name, order_index, created_at
		FROM qa_answer_attachments WHERE id = $1 AND answer_id = $2`, id, parentID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, classroom.ErrAttachmentNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}
