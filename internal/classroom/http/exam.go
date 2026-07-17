package http

import (
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/classroom"
	"github.com/randdotdev/e-campus-server/internal/shared/middleware"
	"github.com/randdotdev/e-campus-server/internal/shared/response"
)

type ExamResponse struct {
	ID               uuid.UUID       `json:"id"`
	OfferingID       uuid.UUID       `json:"offering_id"`
	SectionID        *uuid.UUID      `json:"section_id"`
	Title            string          `json:"title"`
	Description      *string         `json:"description"`
	Type             string          `json:"type"`
	Mode             string          `json:"mode"`
	Questions        json.RawMessage `json:"questions,omitempty"`
	TotalScore       float64         `json:"total_score"`
	DurationMinutes  *int            `json:"duration_minutes"`
	ShuffleQuestions bool            `json:"shuffle_questions"`
	ShuffleOptions   bool            `json:"shuffle_options"`
	ShowResults      string          `json:"show_results"`
	MaxAttempts      int             `json:"max_attempts"`
	AvailableFrom    *time.Time      `json:"available_from"`
	AvailableUntil   *time.Time      `json:"available_until"`
	Status           string          `json:"status"`
	PublishedAt      *time.Time      `json:"published_at"`
	Version          int64           `json:"version"`
	CreatedAt        time.Time       `json:"created_at"`
}

// examResponse blanks the embedded question weights for student views —
// students get questions only through the stripped questions endpoint.
func examResponse(e *classroom.Exam, forStudent bool) ExamResponse {
	resp := ExamResponse{
		ID: e.ID, OfferingID: e.OfferingID, SectionID: e.SectionID,
		Title: e.Title, Description: e.Description, Type: string(e.Type),
		Mode: string(e.Mode), TotalScore: e.TotalScore,
		DurationMinutes: e.DurationMinutes, ShuffleQuestions: e.ShuffleQuestions,
		ShuffleOptions: e.ShuffleOptions, ShowResults: string(e.ShowResults),
		MaxAttempts: e.MaxAttempts, AvailableFrom: e.AvailableFrom,
		AvailableUntil: e.AvailableUntil, Status: string(e.Status),
		PublishedAt: e.PublishedAt, Version: e.Version, CreatedAt: e.CreatedAt,
	}
	if !forStudent {
		resp.Questions = e.Questions
	}
	return resp
}

type AttemptResponse struct {
	ID           uuid.UUID       `json:"id"`
	ExamID       uuid.UUID       `json:"exam_id"`
	StudentID    uuid.UUID       `json:"student_id"`
	Answers      json.RawMessage `json:"answers,omitempty"`
	Scores       json.RawMessage `json:"scores,omitempty"`
	TotalScore   *float64        `json:"total_score"`
	StartedAt    *time.Time      `json:"started_at"`
	SubmittedAt  *time.Time      `json:"submitted_at"`
	LateAccepted *bool           `json:"late_accepted"`
	GradedAt     *time.Time      `json:"graded_at"`
	Visibility   string          `json:"visibility"`
	VisibleAt    *time.Time      `json:"visible_at"`
}

func attemptResponse(a *classroom.Attempt) AttemptResponse {
	return AttemptResponse{
		ID: a.ID, ExamID: a.ExamID, StudentID: a.StudentID,
		Answers: a.Answers, Scores: a.Scores, TotalScore: a.TotalScore,
		StartedAt: a.StartedAt, SubmittedAt: a.SubmittedAt,
		LateAccepted: a.LateAccepted, GradedAt: a.GradedAt,
		Visibility: string(a.Visibility), VisibleAt: a.VisibleAt,
	}
}

// PublicQuestionResponse is the student-facing question: no answer key.
type PublicQuestionResponse struct {
	ID      uuid.UUID       `json:"id"`
	Text    string          `json:"text"`
	ImageID *uuid.UUID      `json:"image_id"`
	Type    string          `json:"type"`
	Options json.RawMessage `json:"options"`
	Score   float64         `json:"score"`
}

type ExamQuestionRef struct {
	ID    uuid.UUID `json:"id" binding:"required"`
	Score float64   `json:"score" binding:"required,gt=0"`
}

type CreateExamRequest struct {
	SectionID        *uuid.UUID        `json:"section_id"`
	Title            string            `json:"title" binding:"required,max=255"`
	Description      *string           `json:"description"`
	Type             string            `json:"type" binding:"required,oneof=exam quiz"`
	Mode             string            `json:"mode" binding:"omitempty,oneof=online manual"`
	Questions        []ExamQuestionRef `json:"questions" binding:"omitempty,dive"`
	DurationMinutes  *int              `json:"duration_minutes" binding:"omitempty,gt=0"`
	ShuffleQuestions bool              `json:"shuffle_questions"`
	ShuffleOptions   bool              `json:"shuffle_options"`
	ShowResults      string            `json:"show_results" binding:"omitempty,oneof=after_submit after_close manual"`
	MaxAttempts      int               `json:"max_attempts" binding:"omitempty,gte=1"`
	AvailableFrom    *time.Time        `json:"available_from"`
	AvailableUntil   *time.Time        `json:"available_until"`
}

type UpdateExamRequest struct {
	SectionID        *uuid.UUID        `json:"section_id"`
	Title            *string           `json:"title" binding:"omitempty,max=255"`
	Description      *string           `json:"description"`
	Questions        []ExamQuestionRef `json:"questions" binding:"omitempty,dive"`
	DurationMinutes  *int              `json:"duration_minutes" binding:"omitempty,gt=0"`
	ShuffleQuestions *bool             `json:"shuffle_questions"`
	ShuffleOptions   *bool             `json:"shuffle_options"`
	ShowResults      *string           `json:"show_results" binding:"omitempty,oneof=after_submit after_close manual"`
	MaxAttempts      *int              `json:"max_attempts" binding:"omitempty,gte=1"`
	AvailableFrom    *time.Time        `json:"available_from"`
	AvailableUntil   *time.Time        `json:"available_until"`
}

func examQuestions(refs []ExamQuestionRef) []classroom.ExamQuestion {
	questions := make([]classroom.ExamQuestion, len(refs))
	for i, ref := range refs {
		questions[i] = classroom.ExamQuestion{ID: ref.ID, Score: ref.Score}
	}
	return questions
}

func (h *Handler) CreateExam(c *gin.Context) {
	var req CreateExamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	e, err := h.exams.Create(c.Request.Context(), classroom.CreateExamInput{
		OfferingID: offeringID(c), CreatedBy: middleware.GetUserID(c),
		SectionID: req.SectionID, Title: req.Title, Description: req.Description,
		Type: classroom.ExamType(req.Type), Mode: classroom.ExamMode(req.Mode),
		Questions: examQuestions(req.Questions), DurationMinutes: req.DurationMinutes,
		ShuffleQuestions: req.ShuffleQuestions, ShuffleOptions: req.ShuffleOptions,
		ShowResults: classroom.ShowResults(req.ShowResults), MaxAttempts: req.MaxAttempts,
		AvailableFrom: req.AvailableFrom, AvailableUntil: req.AvailableUntil,
	})
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Created(c, examResponse(e, false))
}

func (h *Handler) GetExam(c *gin.Context) {
	e, err := h.exams.Get(c.Request.Context(), offeringID(c), targetID(c), studentView(c))
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, examResponse(e, studentView(c)))
}

func (h *Handler) ListExams(c *gin.Context) {
	exams, err := h.exams.List(c.Request.Context(), offeringID(c), studentView(c))
	if err != nil {
		h.respondError(c, err)
		return
	}
	forStudent := studentView(c)
	result := make([]ExamResponse, len(exams))
	for i := range exams {
		result[i] = examResponse(&exams[i], forStudent)
	}
	response.OK(c, result)
}

func (h *Handler) UpdateExam(c *gin.Context) {
	var req UpdateExamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	in := classroom.UpdateExamInput{
		SectionID: req.SectionID, Title: req.Title, Description: req.Description,
		DurationMinutes: req.DurationMinutes, ShuffleQuestions: req.ShuffleQuestions,
		ShuffleOptions: req.ShuffleOptions,
		MaxAttempts:    req.MaxAttempts, AvailableFrom: req.AvailableFrom,
		AvailableUntil: req.AvailableUntil,
	}
	if req.ShowResults != nil {
		sr := classroom.ShowResults(*req.ShowResults)
		in.ShowResults = &sr
	}
	if req.Questions != nil {
		in.Questions = examQuestions(req.Questions)
	}
	e, err := h.exams.Update(c.Request.Context(), offeringID(c), targetID(c), in)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, examResponse(e, false))
}

func (h *Handler) DeleteExam(c *gin.Context) {
	if err := h.exams.Delete(c.Request.Context(), offeringID(c), targetID(c)); err != nil {
		h.respondError(c, err)
		return
	}
	response.NoContent(c)
}

// ExamCustom dispatches the exam's colon methods.
func (h *Handler) ExamCustom(c *gin.Context) {
	ctx := c.Request.Context()
	switch customAction(c) {
	case "publish":
		if err := h.exams.Publish(ctx, offeringID(c), targetID(c)); err != nil {
			h.respondError(c, err)
			return
		}
		response.OK(c, gin.H{"status": "published"})
	case "close":
		if err := h.exams.Close(ctx, offeringID(c), targetID(c)); err != nil {
			h.respondError(c, err)
			return
		}
		response.OK(c, gin.H{"status": "closed"})
	case "start":
		attempt, err := h.exams.Start(ctx, offeringID(c), targetID(c), middleware.GetUserID(c))
		if err != nil {
			h.respondError(c, err)
			return
		}
		response.Created(c, attemptResponse(attempt))
	case "record":
		var req struct {
			Results []struct {
				StudentID  uuid.UUID `json:"student_id" binding:"required"`
				TotalScore float64   `json:"total_score"`
			} `json:"results" binding:"required,min=1,dive"`
			Visibility string     `json:"visibility" binding:"omitempty,oneof=private public scheduled"`
			VisibleAt  *time.Time `json:"visible_at"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		results := make([]classroom.ManualResult, len(req.Results))
		for i, r := range req.Results {
			results[i] = classroom.ManualResult{StudentID: r.StudentID, TotalScore: r.TotalScore}
		}
		if err := h.exams.RecordResults(ctx, offeringID(c), targetID(c), results,
			classroom.ResultVisibility(req.Visibility), req.VisibleAt); err != nil {
			h.respondError(c, err)
			return
		}
		response.OK(c, gin.H{"recorded": len(results)})
	default:
		response.NotFound(c, "unknown method")
	}
}

// ExamQuestions returns the exam's questions; the answer key is stripped
// for student views.
func (h *Handler) ExamQuestions(c *gin.Context) {
	forStudent := studentView(c)
	questions, weights, err := h.exams.QuestionsFor(c.Request.Context(), offeringID(c), targetID(c), forStudent)
	if err != nil {
		h.respondError(c, err)
		return
	}
	if forStudent {
		result := make([]PublicQuestionResponse, len(questions))
		for i, q := range questions {
			result[i] = PublicQuestionResponse{
				ID: q.ID, Text: q.Text, ImageID: q.ImageID,
				Type: string(q.Type), Options: q.Options, Score: weights[q.ID],
			}
		}
		response.OK(c, result)
		return
	}
	result := make([]QuestionResponse, len(questions))
	for i := range questions {
		result[i] = questionResponse(&questions[i])
	}
	response.OK(c, result)
}

// MyAttempt is the caller's own attempt on this exam.
func (h *Handler) MyAttempt(c *gin.Context) {
	attempt, err := h.exams.MyAttempt(c.Request.Context(), offeringID(c), targetID(c), middleware.GetUserID(c))
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, attemptResponse(attempt))
}

func (h *Handler) ListAttempts(c *gin.Context) {
	if !requireTeaching(c) {
		return
	}
	attempts, err := h.exams.Attempts(c.Request.Context(), offeringID(c), targetID(c))
	if err != nil {
		h.respondError(c, err)
		return
	}
	type row struct {
		AttemptResponse
		StudentName  string `json:"student_name"`
		StudentEmail string `json:"student_email"`
	}
	result := make([]row, len(attempts))
	for i, a := range attempts {
		result[i] = row{attemptResponse(&a.Attempt), a.StudentName, a.StudentEmail}
	}
	response.OK(c, result)
}

func (h *Handler) GetAttempt(c *gin.Context) {
	if !requireTeaching(c) {
		return
	}
	attempt, err := h.exams.GetAttempt(c.Request.Context(), offeringID(c), targetID(c))
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, attemptResponse(attempt))
}

// AttemptCustom dispatches the attempt's colon methods.
func (h *Handler) AttemptCustom(c *gin.Context) {
	ctx := c.Request.Context()
	switch customAction(c) {
	case "save":
		var req struct {
			Answers map[string]any `json:"answers" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		if err := h.exams.SaveAnswers(ctx, offeringID(c), targetID(c), middleware.GetUserID(c), req.Answers); err != nil {
			h.respondError(c, err)
			return
		}
		response.OK(c, gin.H{"saved": true})
	case "submit":
		var req struct {
			Answers map[string]any `json:"answers" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		attempt, err := h.exams.Submit(ctx, offeringID(c), targetID(c), middleware.GetUserID(c), req.Answers)
		if err != nil {
			h.respondError(c, err)
			return
		}
		response.OK(c, attemptResponse(attempt))
	case "grade":
		var req struct {
			Scores map[string]float64 `json:"scores" binding:"required,min=1"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		attempt, err := h.exams.Grade(ctx, offeringID(c), targetID(c), middleware.GetUserID(c), req.Scores)
		if err != nil {
			h.respondError(c, err)
			return
		}
		response.OK(c, attemptResponse(attempt))
	case "review":
		var req struct {
			Visibility   *string    `json:"visibility" binding:"omitempty,oneof=private public scheduled"`
			VisibleAt    *time.Time `json:"visible_at"`
			LateAccepted *bool      `json:"late_accepted"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		in := classroom.ReviewAttemptInput{VisibleAt: req.VisibleAt, LateAccepted: req.LateAccepted}
		if req.Visibility != nil {
			v := classroom.ResultVisibility(*req.Visibility)
			in.Visibility = &v
		}
		if err := h.exams.Review(ctx, offeringID(c), targetID(c), in); err != nil {
			h.respondError(c, err)
			return
		}
		response.OK(c, gin.H{"updated": true})
	default:
		response.NotFound(c, "unknown method")
	}
}
