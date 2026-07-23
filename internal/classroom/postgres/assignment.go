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

// AssignmentRepository is the SQL adapter for assignments, their
// attachments, and student submissions.
type AssignmentRepository struct {
	db *sqlx.DB
}

func NewAssignmentRepository(db *sqlx.DB) *AssignmentRepository {
	return &AssignmentRepository{db: db}
}

var _ classroom.AssignmentRepository = (*AssignmentRepository)(nil)

func (r *AssignmentRepository) CreateAssignment(ctx context.Context, a *classroom.Assignment) error {
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO assignments (id, offering_id, title, body, type, deadline, max_score,
			allow_late, publish_at, scores_public, created_by, created_at)
		VALUES (:id, :offering_id, :title, :body, :type, :deadline, :max_score,
			:allow_late, :publish_at, :scores_public, :created_by, :created_at)`, a)
	return err
}

func (r *AssignmentRepository) GetAssignment(ctx context.Context, offeringID, id uuid.UUID) (*classroom.Assignment, error) {
	var a classroom.Assignment
	err := r.db.GetContext(ctx, &a,
		`SELECT * FROM assignments WHERE id = $1 AND offering_id = $2`, id, offeringID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, classroom.ErrAssignmentNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *AssignmentRepository) ListAssignments(ctx context.Context, offeringID uuid.UUID, publishedOnly bool) ([]classroom.Assignment, error) {
	assignments := []classroom.Assignment{}
	query := `SELECT * FROM assignments WHERE offering_id = $1`
	if publishedOnly {
		query += ` AND (publish_at IS NULL OR publish_at <= NOW())`
	}
	query += ` ORDER BY deadline`
	err := r.db.SelectContext(ctx, &assignments, query, offeringID)
	return assignments, err
}

func (r *AssignmentRepository) UpdateAssignment(ctx context.Context, a *classroom.Assignment, expectedVersion int64) (int64, error) {
	return scanVersion(r.db.QueryRowxContext(ctx, `
		UPDATE assignments SET
			title = $1, body = $2, type = $3, deadline = $4, max_score = $5,
			allow_late = $6, publish_at = $7, scores_public = $8, version = version + 1
		WHERE id = $9 AND version = $10
		RETURNING version`,
		a.Title, a.Body, a.Type, a.Deadline, a.MaxScore,
		a.AllowLate, a.PublishAt, a.ScoresPublic,
		a.ID, expectedVersion))
}

// DeleteAssignment removes the assignment; attachments, submissions, and
// submission files ride the FK cascade. Every referenced inode is
// collected first, in the same transaction, for unlinking.
func (r *AssignmentRepository) DeleteAssignment(ctx context.Context, offeringID, id uuid.UUID) ([]uuid.UUID, error) {
	var inodeIDs []uuid.UUID
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	if err := tx.SelectContext(ctx, &inodeIDs, `
		SELECT inode_id FROM assignment_attachments WHERE assignment_id = $1
		UNION ALL
		SELECT sf.inode_id
		FROM submission_files sf
		JOIN assignment_submissions asub ON asub.id = sf.submission_id
		WHERE asub.assignment_id = $1`, id); err != nil {
		return nil, err
	}
	result, err := tx.ExecContext(ctx,
		`DELETE FROM assignments WHERE id = $1 AND offering_id = $2`, id, offeringID)
	if err != nil {
		return nil, err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return nil, classroom.ErrAssignmentNotFound
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return inodeIDs, nil
}

func (r *AssignmentRepository) CreateAttachment(ctx context.Context, a *classroom.AssignmentAttachment) error {
	err := r.db.QueryRowxContext(ctx, `
		INSERT INTO assignment_attachments (id, assignment_id, inode_id, display_name, order_index, added_by)
		VALUES ($1, $2, $3, $4,
			(SELECT COALESCE(MAX(order_index), -1) + 1 FROM assignment_attachments WHERE assignment_id = $2), $5)
		RETURNING order_index`,
		a.ID, a.AssignmentID, a.InodeID, a.DisplayName, a.AddedBy,
	).Scan(&a.OrderIndex)
	return err
}

func (r *AssignmentRepository) ListAttachments(ctx context.Context, assignmentID uuid.UUID) ([]classroom.AssignmentAttachment, error) {
	attachments := []classroom.AssignmentAttachment{}
	err := r.db.SelectContext(ctx, &attachments,
		`SELECT * FROM assignment_attachments WHERE assignment_id = $1 ORDER BY order_index`, assignmentID)
	return attachments, err
}

func (r *AssignmentRepository) GetAttachment(ctx context.Context, assignmentID, id uuid.UUID) (*classroom.AssignmentAttachment, error) {
	var a classroom.AssignmentAttachment
	err := r.db.GetContext(ctx, &a,
		`SELECT * FROM assignment_attachments WHERE id = $1 AND assignment_id = $2`, id, assignmentID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, classroom.ErrAttachmentNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *AssignmentRepository) DeleteAttachment(ctx context.Context, assignmentID, id uuid.UUID) (uuid.UUID, error) {
	var inodeID uuid.UUID
	err := r.db.QueryRowxContext(ctx, `
		DELETE FROM assignment_attachments WHERE id = $1 AND assignment_id = $2
		RETURNING inode_id`, id, assignmentID,
	).Scan(&inodeID)
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, classroom.ErrAttachmentNotFound
	}
	return inodeID, err
}

// SaveDraft upserts the student's draft and replaces its files, one
// transaction. The "still a draft" guard is the upsert's WHERE: a
// submitted or graded row refuses the write.
func (r *AssignmentRepository) SaveDraft(ctx context.Context, sub *classroom.Submission, files []classroom.SubmissionFile) ([]uuid.UUID, error) {
	var replaced []uuid.UUID
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	var subID uuid.UUID
	err = tx.QueryRowxContext(ctx, `
		INSERT INTO assignment_submissions (id, assignment_id, student_id, content, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (assignment_id, student_id) DO UPDATE
			SET content = EXCLUDED.content, updated_at = NOW()
			WHERE assignment_submissions.submitted_at IS NULL
			  AND assignment_submissions.graded_at IS NULL
		RETURNING id`,
		sub.ID, sub.AssignmentID, sub.StudentID, sub.Content, sub.CreatedAt,
	).Scan(&subID)
	if errors.Is(err, sql.ErrNoRows) {
		// The upsert's guard refused: the existing row is past draft.
		return nil, classroom.ErrAlreadySubmitted
	}
	if err != nil {
		return nil, err
	}

	if err := tx.SelectContext(ctx, &replaced,
		`DELETE FROM submission_files WHERE submission_id = $1 RETURNING inode_id`, subID); err != nil {
		return nil, err
	}
	for i := range files {
		files[i].SubmissionID = subID
		if _, err := tx.NamedExecContext(ctx, `
			INSERT INTO submission_files (id, submission_id, inode_id, display_name, order_index, created_at)
			VALUES (:id, :submission_id, :inode_id, :display_name, :order_index, :created_at)`,
			files[i]); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return replaced, nil
}

func (r *AssignmentRepository) SubmitDraft(ctx context.Context, assignmentID, studentID uuid.UUID, at time.Time) (*classroom.Submission, error) {
	var sub classroom.Submission
	err := r.db.QueryRowxContext(ctx, `
		UPDATE assignment_submissions
		SET submitted_at = $1, updated_at = $1
		WHERE assignment_id = $2 AND student_id = $3 AND submitted_at IS NULL
		RETURNING *`, at, assignmentID, studentID,
	).StructScan(&sub)
	if errors.Is(err, sql.ErrNoRows) {
		if _, gerr := r.GetSubmission(ctx, assignmentID, studentID); gerr != nil {
			return nil, gerr
		}
		return nil, classroom.ErrAlreadySubmitted
	}
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

func (r *AssignmentRepository) DiscardDraft(ctx context.Context, assignmentID, studentID uuid.UUID) ([]uuid.UUID, error) {
	var inodeIDs []uuid.UUID
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	if err := tx.SelectContext(ctx, &inodeIDs, `
		SELECT sf.inode_id
		FROM submission_files sf
		JOIN assignment_submissions asub ON asub.id = sf.submission_id
		WHERE asub.assignment_id = $1 AND asub.student_id = $2`, assignmentID, studentID); err != nil {
		return nil, err
	}
	result, err := tx.ExecContext(ctx, `
		DELETE FROM assignment_submissions
		WHERE assignment_id = $1 AND student_id = $2 AND submitted_at IS NULL`,
		assignmentID, studentID)
	if err != nil {
		return nil, err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		if _, gerr := r.GetSubmission(ctx, assignmentID, studentID); gerr != nil {
			return nil, gerr
		}
		return nil, classroom.ErrAlreadySubmitted
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return inodeIDs, nil
}

// GradeSubmission stamps the grade only on a submitted row; grading a
// draft (or nothing) is ErrSubmissionNotFound.
func (r *AssignmentRepository) GradeSubmission(ctx context.Context, assignmentID, studentID, gradedBy uuid.UUID, score float64, feedback *string) (*classroom.Submission, error) {
	var sub classroom.Submission
	err := r.db.QueryRowxContext(ctx, `
		UPDATE assignment_submissions
		SET score = $1, feedback = $2, graded_by = $3, graded_at = NOW(), updated_at = NOW()
		WHERE assignment_id = $4 AND student_id = $5 AND submitted_at IS NOT NULL
		RETURNING *`, score, feedback, gradedBy, assignmentID, studentID,
	).StructScan(&sub)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, classroom.ErrSubmissionNotFound
	}
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

func (r *AssignmentRepository) GetSubmission(ctx context.Context, assignmentID, studentID uuid.UUID) (*classroom.Submission, error) {
	var sub classroom.Submission
	err := r.db.GetContext(ctx, &sub,
		`SELECT * FROM assignment_submissions WHERE assignment_id = $1 AND student_id = $2`,
		assignmentID, studentID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, classroom.ErrSubmissionNotFound
	}
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

func (r *AssignmentRepository) ListSubmissions(ctx context.Context, assignmentID uuid.UUID) ([]classroom.SubmissionWithStudent, error) {
	subs := []classroom.SubmissionWithStudent{}
	err := r.db.SelectContext(ctx, &subs, `
		SELECT asub.*, u.full_name_en AS student_name, u.email AS student_email,
		       u.avatar_url AS student_avatar
		FROM assignment_submissions asub
		JOIN users u ON u.id = asub.student_id
		WHERE asub.assignment_id = $1
		ORDER BY asub.submitted_at NULLS LAST, asub.created_at`, assignmentID)
	return subs, err
}

func (r *AssignmentRepository) ListSubmissionFiles(ctx context.Context, submissionID uuid.UUID) ([]classroom.SubmissionFile, error) {
	files := []classroom.SubmissionFile{}
	err := r.db.SelectContext(ctx, &files,
		`SELECT * FROM submission_files WHERE submission_id = $1 ORDER BY order_index`, submissionID)
	return files, err
}

func (r *AssignmentRepository) GetSubmissionFile(ctx context.Context, submissionID, id uuid.UUID) (*classroom.SubmissionFile, error) {
	var f classroom.SubmissionFile
	err := r.db.GetContext(ctx, &f,
		`SELECT * FROM submission_files WHERE id = $1 AND submission_id = $2`, id, submissionID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, classroom.ErrAttachmentNotFound
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}
