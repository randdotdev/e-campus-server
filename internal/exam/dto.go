package exam

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Filters

type QuestionFilters struct {
	CourseCode *string
	Type       *string
	Difficulty *string
	IsActive   *bool
	CreatedBy  *uuid.UUID
	Query      string
}

type ExamFilters struct {
	OfferingID *uuid.UUID
	SectionID  *uuid.UUID
	Type       *string
	Mode       *string
	Status     *string
}

type AttemptFilters struct {
	ExamID      *uuid.UUID
	StudentID   *uuid.UUID
	IsSubmitted *bool
	IsGraded    *bool
	IsLate      *bool
	Query       string
}

// Request DTOs

type CreateQuestionRequest struct {
	CourseCode   string   `json:"course_code" binding:"required,min=1,max=50"`
	Text         string   `json:"text" binding:"required,min=1"`
	ImageURL     *string  `json:"image_url" binding:"omitempty,url,max=500"`
	Type         string   `json:"type" binding:"required,oneof=single multiple true_false short_answer"`
	Options      []string `json:"options"`
	Correct      any      `json:"correct"`
	DefaultScore float64  `json:"default_score" binding:"omitempty,gt=0"`
	Difficulty   *string  `json:"difficulty" binding:"omitempty,oneof=easy medium hard"`
}

type UpdateQuestionRequest struct {
	Text         *string  `json:"text" binding:"omitempty,min=1"`
	ImageURL     *string  `json:"image_url" binding:"omitempty,url,max=500"`
	Options      []string `json:"options"`
	Correct      any      `json:"correct"`
	DefaultScore *float64 `json:"default_score" binding:"omitempty,gt=0"`
	Difficulty   *string  `json:"difficulty" binding:"omitempty,oneof=easy medium hard"`
}

type BulkCreateQuestionsRequest struct {
	CourseCode string             `json:"course_code" binding:"required,min=1,max=50"`
	Questions  []BulkQuestionItem `json:"questions" binding:"required,min=1,dive"`
}

type BulkQuestionItem struct {
	Text         string   `json:"text" binding:"required,min=1"`
	ImageURL     *string  `json:"image_url" binding:"omitempty,url,max=500"`
	Type         string   `json:"type" binding:"required,oneof=single multiple true_false short_answer"`
	Options      []string `json:"options"`
	Correct      any      `json:"correct"`
	DefaultScore float64  `json:"default_score" binding:"omitempty,gt=0"`
	Difficulty   *string  `json:"difficulty" binding:"omitempty,oneof=easy medium hard"`
}

type RandomSelectRequest struct {
	CourseCode string `form:"course_code" binding:"required"`
	Easy       int    `form:"easy" binding:"min=0"`
	Medium     int    `form:"medium" binding:"min=0"`
	Hard       int    `form:"hard" binding:"min=0"`
}

type CreateExamRequest struct {
	OfferingID       uuid.UUID      `json:"offering_id"`
	SectionID        *uuid.UUID     `json:"section_id"`
	Title            string         `json:"title" binding:"required,min=1,max=255"`
	Description      *string        `json:"description"`
	Type             string         `json:"type" binding:"required,oneof=exam quiz"`
	Mode             string         `json:"mode" binding:"omitempty,oneof=online manual"`
	Questions        []ExamQuestion `json:"questions"`
	DurationMinutes  *int           `json:"duration_minutes" binding:"omitempty,gt=0"`
	ShuffleQuestions bool           `json:"shuffle_questions"`
	ShuffleOptions   bool           `json:"shuffle_options"`
	ShowResults      string         `json:"show_results" binding:"omitempty,oneof=immediately after_submit after_deadline manual"`
	MaxAttempts      int            `json:"max_attempts" binding:"omitempty,min=1"`
	AvailableFrom    *time.Time     `json:"available_from"`
	AvailableUntil   *time.Time     `json:"available_until"`
}

type UpdateExamRequest struct {
	SectionID        *uuid.UUID     `json:"section_id"`
	Title            *string        `json:"title" binding:"omitempty,min=1,max=255"`
	Description      *string        `json:"description"`
	Questions        []ExamQuestion `json:"questions"`
	DurationMinutes  *int           `json:"duration_minutes" binding:"omitempty,gt=0"`
	ShuffleQuestions *bool          `json:"shuffle_questions"`
	ShuffleOptions   *bool          `json:"shuffle_options"`
	ShowResults      *string        `json:"show_results" binding:"omitempty,oneof=immediately after_submit after_deadline manual"`
	MaxAttempts      *int           `json:"max_attempts" binding:"omitempty,min=1"`
	AvailableFrom    *time.Time     `json:"available_from"`
	AvailableUntil   *time.Time     `json:"available_until"`
}

type SaveAnswersRequest struct {
	Answers map[string]any `json:"answers" binding:"required"`
}

type BulkResultsRequest struct {
	Results    []BulkResultItem `json:"results" binding:"required,min=1,dive"`
	Visibility string           `json:"visibility" binding:"omitempty,oneof=private public scheduled"`
	VisibleAt  *time.Time       `json:"visible_at"`
}

type BulkResultItem struct {
	StudentID  uuid.UUID `json:"student_id" binding:"required"`
	TotalScore float64   `json:"total_score" binding:"min=0"`
}

type LateDecisionRequest struct {
	Accepted bool `json:"accepted"`
}

type GradeShortAnswerRequest struct {
	Scores map[string]float64 `json:"scores" binding:"required"`
}

type SetVisibilityRequest struct {
	Visibility string     `json:"visibility" binding:"required,oneof=private public scheduled"`
	VisibleAt  *time.Time `json:"visible_at"`
}

// Response DTOs

type QuestionResponse struct {
	ID           uuid.UUID `json:"id"`
	CourseCode   string    `json:"course_code"`
	Text         string    `json:"text"`
	ImageURL     *string   `json:"image_url,omitempty"`
	Type         string    `json:"type"`
	Options      []string  `json:"options,omitempty"`
	Correct      any       `json:"correct,omitempty"`
	DefaultScore float64   `json:"default_score"`
	Difficulty   *string   `json:"difficulty,omitempty"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type QuestionPublicResponse struct {
	ID       uuid.UUID `json:"id"`
	Text     string    `json:"text"`
	ImageURL *string   `json:"image_url,omitempty"`
	Type     string    `json:"type"`
	Options  []string  `json:"options,omitempty"`
	Score    float64   `json:"score"`
}

type ExamResponse struct {
	ID               uuid.UUID      `json:"id"`
	OfferingID       uuid.UUID      `json:"offering_id"`
	SectionID        *uuid.UUID     `json:"section_id,omitempty"`
	Title            string         `json:"title"`
	Description      *string        `json:"description,omitempty"`
	Type             string         `json:"type"`
	Mode             string         `json:"mode"`
	Questions        []ExamQuestion `json:"questions"`
	TotalScore       float64        `json:"total_score"`
	DurationMinutes  *int           `json:"duration_minutes,omitempty"`
	ShuffleQuestions bool           `json:"shuffle_questions"`
	ShuffleOptions   bool           `json:"shuffle_options"`
	ShowResults      string         `json:"show_results"`
	MaxAttempts      int            `json:"max_attempts"`
	AvailableFrom    *time.Time     `json:"available_from,omitempty"`
	AvailableUntil   *time.Time     `json:"available_until,omitempty"`
	UsedAt           *time.Time     `json:"used_at,omitempty"`
	Status           string         `json:"status"`
	PublishedAt      *time.Time     `json:"published_at,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
}

type ExamListResponse struct {
	ID              uuid.UUID  `json:"id"`
	OfferingID      uuid.UUID  `json:"offering_id"`
	SectionID       *uuid.UUID `json:"section_id,omitempty"`
	Title           string     `json:"title"`
	Type            string     `json:"type"`
	Mode            string     `json:"mode"`
	TotalScore      float64    `json:"total_score"`
	QuestionCount   int        `json:"question_count"`
	DurationMinutes *int       `json:"duration_minutes,omitempty"`
	MaxAttempts     int        `json:"max_attempts"`
	AvailableFrom   *time.Time `json:"available_from,omitempty"`
	AvailableUntil  *time.Time `json:"available_until,omitempty"`
	Status          string     `json:"status"`
	CreatedAt       time.Time  `json:"created_at"`
}

type AttemptResponse struct {
	ID          uuid.UUID           `json:"id"`
	ExamID      uuid.UUID           `json:"exam_id"`
	StudentID   uuid.UUID           `json:"student_id"`
	Answers     map[string]any      `json:"answers,omitempty"`
	Scores      map[string]*float64 `json:"scores,omitempty"`
	TotalScore  *float64            `json:"total_score,omitempty"`
	StartedAt   *time.Time          `json:"started_at,omitempty"`
	SubmittedAt *time.Time          `json:"submitted_at,omitempty"`
	IsLate      bool                `json:"is_late"`
	GradedAt    *time.Time          `json:"graded_at,omitempty"`
	Visibility  string              `json:"visibility"`
	VisibleAt   *time.Time          `json:"visible_at,omitempty"`
}

type AttemptListResponse struct {
	ID          uuid.UUID  `json:"id"`
	ExamID      uuid.UUID  `json:"exam_id"`
	StudentID   uuid.UUID  `json:"student_id"`
	StudentName string     `json:"student_name,omitempty"`
	TotalScore  *float64   `json:"total_score,omitempty"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	SubmittedAt *time.Time `json:"submitted_at,omitempty"`
	IsLate      bool       `json:"is_late"`
	IsGraded    bool       `json:"is_graded"`
	Visibility  string     `json:"visibility"`
}

type BulkCreateResponse struct {
	Created int `json:"created"`
	Skipped int `json:"skipped"`
}

type RandomSelectResponse struct {
	Questions []QuestionResponse `json:"questions"`
	Warnings  []string           `json:"warnings,omitempty"`
}

// Mappers

func ToQuestionResponse(q *Question, includeCorrect bool) QuestionResponse {
	var options []string
	if q.Options != nil {
		_ = json.Unmarshal(q.Options, &options)
	}

	var correct any
	if includeCorrect && q.Correct != nil {
		_ = json.Unmarshal(q.Correct, &correct)
	}

	return QuestionResponse{
		ID:           q.ID,
		CourseCode:   q.CourseCode,
		Text:         q.Text,
		ImageURL:     q.ImageURL,
		Type:         q.Type,
		Options:      options,
		Correct:      correct,
		DefaultScore: q.DefaultScore,
		Difficulty:   q.Difficulty,
		IsActive:     q.IsActive,
		CreatedAt:    q.CreatedAt,
		UpdatedAt:    q.UpdatedAt,
	}
}

func ToQuestionsResponse(questions []Question, includeCorrect bool) []QuestionResponse {
	result := make([]QuestionResponse, len(questions))
	for i := range questions {
		result[i] = ToQuestionResponse(&questions[i], includeCorrect)
	}
	return result
}

func ToQuestionPublicResponse(q *Question, score float64) QuestionPublicResponse {
	var options []string
	if q.Options != nil {
		_ = json.Unmarshal(q.Options, &options)
	}

	return QuestionPublicResponse{
		ID:       q.ID,
		Text:     q.Text,
		ImageURL: q.ImageURL,
		Type:     q.Type,
		Options:  options,
		Score:    score,
	}
}

func ToExamResponse(e *Exam) ExamResponse {
	var questions []ExamQuestion
	if e.Questions != nil {
		_ = json.Unmarshal(e.Questions, &questions)
	}

	return ExamResponse{
		ID:               e.ID,
		OfferingID:       e.OfferingID,
		SectionID:        e.SectionID,
		Title:            e.Title,
		Description:      e.Description,
		Type:             e.Type,
		Mode:             e.Mode,
		Questions:        questions,
		TotalScore:       e.TotalScore,
		DurationMinutes:  e.DurationMinutes,
		ShuffleQuestions: e.ShuffleQuestions,
		ShuffleOptions:   e.ShuffleOptions,
		ShowResults:      e.ShowResults,
		MaxAttempts:      e.MaxAttempts,
		AvailableFrom:    e.AvailableFrom,
		AvailableUntil:   e.AvailableUntil,
		UsedAt:           e.UsedAt,
		Status:           e.Status,
		PublishedAt:      e.PublishedAt,
		CreatedAt:        e.CreatedAt,
	}
}

func ToExamListResponse(e *Exam) ExamListResponse {
	var questions []ExamQuestion
	if e.Questions != nil {
		_ = json.Unmarshal(e.Questions, &questions)
	}

	return ExamListResponse{
		ID:              e.ID,
		OfferingID:      e.OfferingID,
		SectionID:       e.SectionID,
		Title:           e.Title,
		Type:            e.Type,
		Mode:            e.Mode,
		TotalScore:      e.TotalScore,
		QuestionCount:   len(questions),
		DurationMinutes: e.DurationMinutes,
		MaxAttempts:     e.MaxAttempts,
		AvailableFrom:   e.AvailableFrom,
		AvailableUntil:  e.AvailableUntil,
		Status:          e.Status,
		CreatedAt:       e.CreatedAt,
	}
}

func ToExamsListResponse(exams []Exam) []ExamListResponse {
	result := make([]ExamListResponse, len(exams))
	for i := range exams {
		result[i] = ToExamListResponse(&exams[i])
	}
	return result
}

func ToAttemptResponse(a *Attempt, deadline *time.Time) AttemptResponse {
	var answers map[string]any
	if a.Answers != nil {
		_ = json.Unmarshal(a.Answers, &answers)
	}

	var scores map[string]*float64
	if a.Scores != nil {
		_ = json.Unmarshal(a.Scores, &scores)
	}

	isLate := false
	if a.SubmittedAt != nil && deadline != nil {
		isLate = a.SubmittedAt.After(*deadline)
	}

	return AttemptResponse{
		ID:          a.ID,
		ExamID:      a.ExamID,
		StudentID:   a.StudentID,
		Answers:     answers,
		Scores:      scores,
		TotalScore:  a.TotalScore,
		StartedAt:   a.StartedAt,
		SubmittedAt: a.SubmittedAt,
		IsLate:      isLate,
		GradedAt:    a.GradedAt,
		Visibility:  a.Visibility,
		VisibleAt:   a.VisibleAt,
	}
}

func ToAttemptListResponse(a *Attempt, deadline *time.Time, studentName string) AttemptListResponse {
	isLate := false
	if a.SubmittedAt != nil && deadline != nil {
		isLate = a.SubmittedAt.After(*deadline)
	}

	return AttemptListResponse{
		ID:          a.ID,
		ExamID:      a.ExamID,
		StudentID:   a.StudentID,
		StudentName: studentName,
		TotalScore:  a.TotalScore,
		StartedAt:   a.StartedAt,
		SubmittedAt: a.SubmittedAt,
		IsLate:      isLate,
		IsGraded:    a.GradedAt != nil,
		Visibility:  a.Visibility,
	}
}
