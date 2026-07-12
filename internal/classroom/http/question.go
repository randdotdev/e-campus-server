package http

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/classroom"
	"github.com/randdotdev/e-campus-server/internal/shared/middleware"
	"github.com/randdotdev/e-campus-server/internal/shared/response"
)

// QuestionResponse is the teacher's view; the answer key stays in. The
// student-facing exam shape (PublicQuestionResponse) never carries it.
type QuestionResponse struct {
	ID           uuid.UUID       `json:"id"`
	CourseCode   string          `json:"course_code"`
	Text         string          `json:"text"`
	ImageID      *uuid.UUID      `json:"image_id"`
	Type         string          `json:"type"`
	Options      json.RawMessage `json:"options"`
	Correct      json.RawMessage `json:"correct"`
	DefaultScore float64         `json:"default_score"`
	Difficulty   *string         `json:"difficulty"`
	CreatedAt    time.Time       `json:"created_at"`
}

func questionResponse(q *classroom.Question) QuestionResponse {
	var difficulty *string
	if q.Difficulty != nil {
		d := string(*q.Difficulty)
		difficulty = &d
	}
	return QuestionResponse{
		ID: q.ID, CourseCode: q.CourseCode, Text: q.Text, ImageID: q.ImageID,
		Type: string(q.Type), Options: q.Options, Correct: q.Correct,
		DefaultScore: q.DefaultScore, Difficulty: difficulty, CreatedAt: q.CreatedAt,
	}
}

type QuestionRequest struct {
	Text         string          `json:"text" binding:"required"`
	ImageFileID  *uuid.UUID      `json:"image_file_id"`
	Type         string          `json:"type" binding:"required,oneof=single multiple true_false short_answer"`
	Options      []string        `json:"options"`
	Correct      json.RawMessage `json:"correct"`
	DefaultScore float64         `json:"default_score" binding:"omitempty,gt=0"`
	Difficulty   *string         `json:"difficulty" binding:"omitempty,oneof=easy medium hard"`
}

func (r *QuestionRequest) toInput() classroom.QuestionInput {
	in := classroom.QuestionInput{
		Text:         r.Text,
		Type:         classroom.QuestionType(r.Type),
		Options:      r.Options,
		DefaultScore: r.DefaultScore,
	}
	if r.Correct != nil {
		var correct any
		_ = json.Unmarshal(r.Correct, &correct) //nolint:errcheck // valid JSON by binding; re-marshalled downstream
		in.Correct = correct
	}
	if r.ImageFileID != nil {
		in.ImageFile = &classroom.FileRef{UploadID: *r.ImageFileID}
	}
	if r.Difficulty != nil {
		d := classroom.Difficulty(*r.Difficulty)
		in.Difficulty = &d
	}
	return in
}

type UpdateQuestionRequest struct {
	Text         *string         `json:"text"`
	ImageFileID  *uuid.UUID      `json:"image_file_id"`
	ClearImage   bool            `json:"clear_image"`
	Options      []string        `json:"options"`
	Correct      json.RawMessage `json:"correct"`
	DefaultScore *float64        `json:"default_score" binding:"omitempty,gt=0"`
	Difficulty   *string         `json:"difficulty" binding:"omitempty,oneof=easy medium hard"`
}

func (h *Handler) CreateQuestion(c *gin.Context) {
	if !requireTeaching(c) {
		return
	}
	var req QuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	q, err := h.questions.Create(c.Request.Context(), offeringID(c), middleware.GetUserID(c), req.toInput())
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Created(c, questionResponse(q))
}

func (h *Handler) BulkCreateQuestions(c *gin.Context) {
	if !requireTeaching(c) {
		return
	}
	var req struct {
		Questions []QuestionRequest `json:"questions" binding:"required,min=1,max=200,dive"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	inputs := make([]classroom.QuestionInput, len(req.Questions))
	for i := range req.Questions {
		inputs[i] = req.Questions[i].toInput()
	}
	created, skipped, err := h.questions.CreateBulk(c.Request.Context(), offeringID(c), middleware.GetUserID(c), inputs)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, gin.H{"created": created, "skipped": skipped})
}

func (h *Handler) GetQuestion(c *gin.Context) {
	if !requireTeaching(c) {
		return
	}
	q, err := h.questions.Get(c.Request.Context(), offeringID(c), targetID(c))
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, questionResponse(q))
}

// ListQuestions is teacher-only: the bank carries answer keys.
func (h *Handler) ListQuestions(c *gin.Context) {
	if !requireTeaching(c) {
		return
	}
	filter := classroom.QuestionFilter{Search: c.Query("q")}
	if v := c.Query("type"); v != "" {
		t := classroom.QuestionType(v)
		filter.Type = &t
	}
	if v := c.Query("difficulty"); v != "" {
		d := classroom.Difficulty(v)
		filter.Difficulty = &d
	}
	questions, err := h.questions.List(c.Request.Context(), offeringID(c), filter)
	if err != nil {
		h.respondError(c, err)
		return
	}
	result := make([]QuestionResponse, len(questions))
	for i := range questions {
		result[i] = questionResponse(&questions[i])
	}
	response.OK(c, result)
}

func (h *Handler) UpdateQuestion(c *gin.Context) {
	if !requireTeaching(c) {
		return
	}
	var req UpdateQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	in := classroom.UpdateQuestionInput{
		Text: req.Text, ClearImage: req.ClearImage,
		Options: req.Options, DefaultScore: req.DefaultScore,
	}
	if req.Correct != nil {
		var correct any
		_ = json.Unmarshal(req.Correct, &correct) //nolint:errcheck // valid JSON by binding; re-marshalled downstream
		in.Correct = correct
	}
	if req.ImageFileID != nil {
		in.ImageFile = &classroom.FileRef{UploadID: *req.ImageFileID}
	}
	if req.Difficulty != nil {
		d := classroom.Difficulty(*req.Difficulty)
		in.Difficulty = &d
	}
	q, err := h.questions.Update(c.Request.Context(), offeringID(c), targetID(c), middleware.GetUserID(c), in)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, questionResponse(q))
}

func (h *Handler) DeactivateQuestion(c *gin.Context) {
	if !requireTeaching(c) {
		return
	}
	if err := h.questions.Deactivate(c.Request.Context(), offeringID(c), targetID(c)); err != nil {
		h.respondError(c, err)
		return
	}
	response.NoContent(c)
}

// SampleQuestions draws a random exam-sized set by difficulty counts.
func (h *Handler) SampleQuestions(c *gin.Context) {
	if !requireTeaching(c) {
		return
	}
	counts := classroom.SampleCounts{
		Easy:   queryInt(c, "easy"),
		Medium: queryInt(c, "medium"),
		Hard:   queryInt(c, "hard"),
	}
	questions, warnings, err := h.questions.Sample(c.Request.Context(), offeringID(c), counts)
	if err != nil {
		h.respondError(c, err)
		return
	}
	result := make([]QuestionResponse, len(questions))
	for i := range questions {
		result[i] = questionResponse(&questions[i])
	}
	response.OK(c, gin.H{"questions": result, "warnings": warnings})
}

func queryInt(c *gin.Context, key string) int {
	n, _ := strconv.Atoi(c.Query(key)) //nolint:errcheck // absent or malformed means zero of that tier
	return n
}
