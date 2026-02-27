package assignment

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

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, a *Assignment) error {
	query := `
		INSERT INTO assignments (id, offering_id, title, body, type, deadline, max_score, allow_late, publish_at, scores_public, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`
	_, err := r.db.ExecContext(ctx, query,
		a.ID, a.OfferingID, a.Title, a.Body, a.Type, a.Deadline, a.MaxScore, a.AllowLate, a.PublishAt, a.ScoresPublic, a.CreatedBy, a.CreatedAt)
	return err
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Assignment, error) {
	var a Assignment
	query := `SELECT * FROM assignments WHERE id = $1`
	if err := r.db.GetContext(ctx, &a, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

func (r *Repository) GetByOffering(ctx context.Context, offeringID uuid.UUID) ([]Assignment, error) {
	var assignments []Assignment
	query := `SELECT * FROM assignments WHERE offering_id = $1 ORDER BY deadline`
	if err := r.db.SelectContext(ctx, &assignments, query, offeringID); err != nil {
		return nil, err
	}
	return assignments, nil
}

func (r *Repository) GetPublishedByOffering(ctx context.Context, offeringID uuid.UUID, now time.Time) ([]Assignment, error) {
	var assignments []Assignment
	query := `SELECT * FROM assignments WHERE offering_id = $1 AND publish_at IS NOT NULL AND publish_at <= $2 ORDER BY deadline`
	if err := r.db.SelectContext(ctx, &assignments, query, offeringID, now); err != nil {
		return nil, err
	}
	return assignments, nil
}

func (r *Repository) Update(ctx context.Context, a *Assignment) error {
	query := `
		UPDATE assignments
		SET title = $1, body = $2, type = $3, deadline = $4, max_score = $5, allow_late = $6, publish_at = $7, scores_public = $8
		WHERE id = $9`
	_, err := r.db.ExecContext(ctx, query,
		a.Title, a.Body, a.Type, a.Deadline, a.MaxScore, a.AllowLate, a.PublishAt, a.ScoresPublic, a.ID)
	return err
}

func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM assignments WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *Repository) CreateAttachment(ctx context.Context, a *AssignmentAttachment) error {
	query := `
		INSERT INTO assignment_attachments (id, assignment_id, stored_file_id, display_name, order_index, added_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.db.ExecContext(ctx, query,
		a.ID, a.AssignmentID, a.StoredFileID, a.DisplayName, a.OrderIndex, a.AddedBy, a.CreatedAt)
	return err
}

func (r *Repository) GetAttachmentByID(ctx context.Context, id uuid.UUID) (*AssignmentAttachment, error) {
	var a AssignmentAttachment
	query := `SELECT * FROM assignment_attachments WHERE id = $1`
	if err := r.db.GetContext(ctx, &a, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

func (r *Repository) GetAttachments(ctx context.Context, assignmentID uuid.UUID) ([]AssignmentAttachment, error) {
	var attachments []AssignmentAttachment
	query := `SELECT * FROM assignment_attachments WHERE assignment_id = $1 ORDER BY order_index`
	if err := r.db.SelectContext(ctx, &attachments, query, assignmentID); err != nil {
		return nil, err
	}
	return attachments, nil
}

func (r *Repository) DeleteAttachment(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM assignment_attachments WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *Repository) CreateSubmission(ctx context.Context, s *Submission) error {
	query := `
		INSERT INTO assignment_submissions (id, assignment_id, student_id, content, submitted_at, score, feedback, graded_by, graded_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`
	_, err := r.db.ExecContext(ctx, query,
		s.ID, s.AssignmentID, s.StudentID, s.Content, s.SubmittedAt, s.Score, s.Feedback, s.GradedBy, s.GradedAt, s.CreatedAt, s.UpdatedAt)
	return err
}

func (r *Repository) GetSubmissionByID(ctx context.Context, id uuid.UUID) (*Submission, error) {
	var s Submission
	query := `SELECT * FROM assignment_submissions WHERE id = $1`
	if err := r.db.GetContext(ctx, &s, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *Repository) GetSubmission(ctx context.Context, assignmentID, studentID uuid.UUID) (*Submission, error) {
	var s Submission
	query := `SELECT * FROM assignment_submissions WHERE assignment_id = $1 AND student_id = $2`
	if err := r.db.GetContext(ctx, &s, query, assignmentID, studentID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *Repository) GetSubmissionsByAssignment(ctx context.Context, assignmentID uuid.UUID) ([]SubmissionWithStudent, error) {
	var submissions []SubmissionWithStudent
	query := `
		SELECT s.*, u.full_name_en as student_name
		FROM assignment_submissions s
		JOIN users u ON s.student_id = u.id
		WHERE s.assignment_id = $1
		ORDER BY u.full_name_en`
	if err := r.db.SelectContext(ctx, &submissions, query, assignmentID); err != nil {
		return nil, err
	}
	return submissions, nil
}

func (r *Repository) UpdateSubmission(ctx context.Context, s *Submission) error {
	query := `
		UPDATE assignment_submissions
		SET content = $1, submitted_at = $2, score = $3, feedback = $4, graded_by = $5, graded_at = $6, updated_at = $7
		WHERE id = $8`
	_, err := r.db.ExecContext(ctx, query,
		s.Content, s.SubmittedAt, s.Score, s.Feedback, s.GradedBy, s.GradedAt, s.UpdatedAt, s.ID)
	return err
}

func (r *Repository) DeleteSubmission(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM assignment_submissions WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *Repository) CreateSubmissionFile(ctx context.Context, f *SubmissionFile) error {
	query := `
		INSERT INTO submission_files (id, submission_id, stored_file_id, display_name, order_index, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.db.ExecContext(ctx, query,
		f.ID, f.SubmissionID, f.StoredFileID, f.DisplayName, f.OrderIndex, f.CreatedAt)
	return err
}

func (r *Repository) GetSubmissionFiles(ctx context.Context, submissionID uuid.UUID) ([]SubmissionFile, error) {
	var files []SubmissionFile
	query := `SELECT * FROM submission_files WHERE submission_id = $1 ORDER BY order_index`
	if err := r.db.SelectContext(ctx, &files, query, submissionID); err != nil {
		return nil, err
	}
	return files, nil
}

func (r *Repository) DeleteSubmissionFiles(ctx context.Context, submissionID uuid.UUID) error {
	query := `DELETE FROM submission_files WHERE submission_id = $1`
	_, err := r.db.ExecContext(ctx, query, submissionID)
	return err
}

func (r *Repository) UserOwnsFiles(ctx context.Context, userID uuid.UUID, storedFileIDs []uuid.UUID) (bool, error) {
	if len(storedFileIDs) == 0 {
		return true, nil
	}

	query := `SELECT COUNT(*) FROM user_files WHERE owner_id = $1 AND stored_file_id = ANY($2)`
	var count int
	if err := r.db.GetContext(ctx, &count, query, userID, storedFileIDs); err != nil {
		return false, err
	}
	return count == len(storedFileIDs), nil
}
