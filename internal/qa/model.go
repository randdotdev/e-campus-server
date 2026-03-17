// Package qa handles student questions and teacher answers.
package qa

import (
	"time"

	"github.com/google/uuid"
)

type Question struct {
	ID          uuid.UUID  `db:"id"`
	OfferingID  uuid.UUID  `db:"offering_id"`
	Title       string     `db:"title"`
	Body        string     `db:"body"`
	IsAnonymous bool       `db:"is_anonymous"`
	IsFAQ       bool       `db:"is_faq"`
	Status      string     `db:"status"`
	CreatedBy   uuid.UUID  `db:"created_by"`
	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   *time.Time `db:"updated_at"`
	EditedBy    *uuid.UUID `db:"edited_by"`
	DeletedAt   *time.Time `db:"deleted_at"`
}

type QuestionWithAuthor struct {
	Question
	AuthorName      string  `db:"author_name"`
	AuthorNameLocal *string `db:"author_name_local"`
}

type Answer struct {
	ID         uuid.UUID  `db:"id"`
	QuestionID uuid.UUID  `db:"question_id"`
	Body       string     `db:"body"`
	CreatedBy  uuid.UUID  `db:"created_by"`
	CreatedAt  time.Time  `db:"created_at"`
	UpdatedBy  *uuid.UUID `db:"updated_by"`
	UpdatedAt  *time.Time `db:"updated_at"`
}

type AnswerWithAuthor struct {
	Answer
	AuthorName      string  `db:"author_name"`
	AuthorNameLocal *string `db:"author_name_local"`
}

type QuestionAttachment struct {
	ID         uuid.UUID `db:"id"`
	QuestionID uuid.UUID `db:"question_id"`
	FilePath   string    `db:"file_path"`
	FileName   string    `db:"file_name"`
	FileSize   int       `db:"file_size"`
	MimeType   string    `db:"mime_type"`
	CreatedAt  time.Time `db:"created_at"`
}

type AnswerAttachment struct {
	ID        uuid.UUID `db:"id"`
	AnswerID  uuid.UUID `db:"answer_id"`
	FilePath  string    `db:"file_path"`
	FileName  string    `db:"file_name"`
	FileSize  int       `db:"file_size"`
	MimeType  string    `db:"mime_type"`
	CreatedAt time.Time `db:"created_at"`
}

type QuestionRejection struct {
	QuestionID uuid.UUID `db:"question_id"`
	Reason     string    `db:"reason"`
	RejectedBy uuid.UUID `db:"rejected_by"`
	RejectedAt time.Time `db:"rejected_at"`
}

type QuestionRejectionWithUser struct {
	QuestionRejection
	RejectedByName      string  `db:"rejected_by_name"`
	RejectedByNameLocal *string `db:"rejected_by_name_local"`
}

const (
	StatusPending  = "pending"
	StatusAnswered = "answered"
	StatusRejected = "rejected"
)
