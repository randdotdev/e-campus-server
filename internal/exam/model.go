// Package exam handles question bank, exams, and exam attempts.
package exam

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Question struct {
	ID           uuid.UUID       `db:"id"`
	CourseCode   string          `db:"course_code"`
	Text         string          `db:"text"`
	ImageURL     *string         `db:"image_url"`
	Type         string          `db:"type"`
	Options      json.RawMessage `db:"options"`
	Correct      json.RawMessage `db:"correct"`
	DefaultScore float64         `db:"default_score"`
	Difficulty   *string         `db:"difficulty"`
	IsActive     bool            `db:"is_active"`
	CreatedBy    *uuid.UUID      `db:"created_by"`
	CreatedAt    time.Time       `db:"created_at"`
	UpdatedAt    time.Time       `db:"updated_at"`
}

type Exam struct {
	ID               uuid.UUID       `db:"id"`
	OfferingID       uuid.UUID       `db:"offering_id"`
	SectionID        *uuid.UUID      `db:"section_id"`
	Title            string          `db:"title"`
	Description      *string         `db:"description"`
	Type             string          `db:"type"`
	Mode             string          `db:"mode"`
	Questions        json.RawMessage `db:"questions"`
	TotalScore       float64         `db:"total_score"`
	DurationMinutes  *int            `db:"duration_minutes"`
	ShuffleQuestions bool            `db:"shuffle_questions"`
	ShuffleOptions   bool            `db:"shuffle_options"`
	ShowResults      string          `db:"show_results"`
	MaxAttempts      int             `db:"max_attempts"`
	AvailableFrom    *time.Time      `db:"available_from"`
	AvailableUntil   *time.Time      `db:"available_until"`
	UsedAt           *time.Time      `db:"used_at"`
	Status           string          `db:"status"`
	PublishedAt      *time.Time      `db:"published_at"`
	CreatedBy        *uuid.UUID      `db:"created_by"`
	CreatedAt        time.Time       `db:"created_at"`
}

type ExamQuestion struct {
	ID    uuid.UUID `json:"id"`
	Score float64   `json:"score"`
}

type Attempt struct {
	ID           uuid.UUID       `db:"id"`
	ExamID       uuid.UUID       `db:"exam_id"`
	StudentID    uuid.UUID       `db:"student_id"`
	Answers      json.RawMessage `db:"answers"`
	Scores       json.RawMessage `db:"scores"`
	TotalScore   *float64        `db:"total_score"`
	StartedAt    *time.Time      `db:"started_at"`
	UpdatedAt    *time.Time      `db:"updated_at"`
	SubmittedAt  *time.Time      `db:"submitted_at"`
	LateAccepted *bool           `db:"late_accepted"`
	GradedBy     *uuid.UUID      `db:"graded_by"`
	GradedAt     *time.Time      `db:"graded_at"`
	Visibility   string          `db:"visibility"`
	VisibleAt    *time.Time      `db:"visible_at"`
}

// Question type constants
const (
	QuestionTypeSingle      = "single"
	QuestionTypeMultiple    = "multiple"
	QuestionTypeTrueFalse   = "true_false"
	QuestionTypeShortAnswer = "short_answer"
)

// Question difficulty constants
const (
	DifficultyEasy   = "easy"
	DifficultyMedium = "medium"
	DifficultyHard   = "hard"
)

// Exam type constants
const (
	ExamTypeExam = "exam"
	ExamTypeQuiz = "quiz"
)

// Exam mode constants
const (
	ExamModeOnline = "online"
	ExamModeManual = "manual"
)

// Exam status constants
const (
	ExamStatusDraft     = "draft"
	ExamStatusPublished = "published"
	ExamStatusClosed    = "closed"
)

// Show results constants
const (
	ShowResultsImmediately   = "immediately"
	ShowResultsAfterSubmit   = "after_submit"
	ShowResultsAfterDeadline = "after_deadline"
	ShowResultsManual        = "manual"
)

// Visibility constants
const (
	VisibilityPrivate   = "private"
	VisibilityPublic    = "public"
	VisibilityScheduled = "scheduled"
)
