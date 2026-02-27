// Package assignment handles homework, projects, and student submissions.
package assignment

import (
	"time"

	"github.com/google/uuid"
)

type Assignment struct {
	ID           uuid.UUID  `db:"id"`
	OfferingID   uuid.UUID  `db:"offering_id"`
	Title        string     `db:"title"`
	Body         *string    `db:"body"`
	Type         *string    `db:"type"`
	Deadline     time.Time  `db:"deadline"`
	MaxScore     float64    `db:"max_score"`
	AllowLate    bool       `db:"allow_late"`
	PublishAt    *time.Time `db:"publish_at"`
	ScoresPublic bool       `db:"scores_public"`
	CreatedBy    *uuid.UUID `db:"created_by"`
	CreatedAt    time.Time  `db:"created_at"`
}

type AssignmentAttachment struct {
	ID           uuid.UUID  `db:"id"`
	AssignmentID uuid.UUID  `db:"assignment_id"`
	StoredFileID uuid.UUID  `db:"stored_file_id"`
	DisplayName  string     `db:"display_name"`
	OrderIndex   int        `db:"order_index"`
	AddedBy      *uuid.UUID `db:"added_by"`
	CreatedAt    time.Time  `db:"created_at"`
}

type Submission struct {
	ID           uuid.UUID  `db:"id"`
	AssignmentID uuid.UUID  `db:"assignment_id"`
	StudentID    uuid.UUID  `db:"student_id"`
	Content      *string    `db:"content"`
	SubmittedAt  *time.Time `db:"submitted_at"`
	Score        *float64   `db:"score"`
	Feedback     *string    `db:"feedback"`
	GradedBy     *uuid.UUID `db:"graded_by"`
	GradedAt     *time.Time `db:"graded_at"`
	CreatedAt    time.Time  `db:"created_at"`
	UpdatedAt    *time.Time `db:"updated_at"`
}

type SubmissionWithStudent struct {
	Submission
	StudentName string `db:"student_name"`
}

type SubmissionFile struct {
	ID           uuid.UUID `db:"id"`
	SubmissionID uuid.UUID `db:"submission_id"`
	StoredFileID uuid.UUID `db:"stored_file_id"`
	DisplayName  string    `db:"display_name"`
	OrderIndex   int       `db:"order_index"`
	CreatedAt    time.Time `db:"created_at"`
}

const (
	TypeTheory   = "theory"
	TypePractice = "practice"

	StatusDraft     = "draft"
	StatusSubmitted = "submitted"
	StatusGraded    = "graded"
)
