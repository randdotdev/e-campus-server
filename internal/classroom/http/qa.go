package http

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/classroom"
	"github.com/randdotdev/e-campus-server/internal/shared/middleware"
	"github.com/randdotdev/e-campus-server/internal/shared/response"
)

type QAQuestionResponse struct {
	ID          uuid.UUID `json:"id"`
	OfferingID  uuid.UUID `json:"offering_id"`
	Title       string    `json:"title"`
	Body        string    `json:"body"`
	IsAnonymous bool      `json:"is_anonymous"`
	IsFAQ       bool      `json:"is_faq"`
	Status      string    `json:"status"`
	// Author fields are blanked when the question is anonymous and the
	// reader is not teaching staff.
	AuthorID       *uuid.UUID `json:"author_id"`
	AuthorName     *string    `json:"author_name"`
	AuthorUsername *string    `json:"author_username"`
	AuthorAvatar   *string    `json:"author_avatar"`
	Version        int64      `json:"version"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      *time.Time `json:"updated_at"`
}

func qaQuestionResponse(q *classroom.QAQuestionView, revealAuthor bool) QAQuestionResponse {
	resp := QAQuestionResponse{
		ID: q.ID, OfferingID: q.OfferingID, Title: q.Title, Body: q.Body,
		IsAnonymous: q.IsAnonymous, IsFAQ: q.IsFAQ, Status: string(q.Status),
		Version: q.Version, CreatedAt: q.CreatedAt, UpdatedAt: q.UpdatedAt,
	}
	if !q.IsAnonymous || revealAuthor {
		resp.AuthorID = &q.CreatedBy
		resp.AuthorName = &q.AuthorName
		resp.AuthorUsername = &q.AuthorUsername
		resp.AuthorAvatar = q.AuthorAvatar
	}
	return resp
}

type QAAnswerResponse struct {
	ID         uuid.UUID  `json:"id"`
	Body       string     `json:"body"`
	AuthorName string     `json:"author_name"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  *time.Time `json:"updated_at"`
}

type AskQuestionRequest struct {
	Title       string           `json:"title" binding:"required,max=255"`
	Body        string           `json:"body" binding:"required,max=10000"`
	IsAnonymous bool             `json:"is_anonymous"`
	Files       []FileRefRequest `json:"files" binding:"omitempty,max=10,dive"`
	// Answer turns the post into an FAQ; teaching staff only.
	Answer      *string          `json:"answer" binding:"omitempty,max=10000"`
	AnswerFiles []FileRefRequest `json:"answer_files" binding:"omitempty,max=10,dive"`
}

func (h *Handler) AskQuestion(c *gin.Context) {
	var req AskQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if req.Answer != nil && !requireTeaching(c) {
		return
	}
	q, err := h.qa.Ask(c.Request.Context(), classroom.AskInput{
		OfferingID:  offeringID(c),
		AuthorID:    middleware.GetUserID(c),
		Title:       req.Title,
		Body:        req.Body,
		IsAnonymous: req.IsAnonymous,
		Files:       fileRefs(req.Files),
		Answer:      req.Answer,
		AnswerFiles: fileRefs(req.AnswerFiles),
	})
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Created(c, gin.H{"id": q.ID, "status": q.Status})
}

func (h *Handler) GetQAQuestion(c *gin.Context) {
	reader := middleware.GetUserID(c)
	isStaff := teaching(c)
	q, answer, qAtts, aAtts, rejection, err := h.qa.Get(c.Request.Context(), offeringID(c), targetID(c), reader, isStaff)
	if err != nil {
		h.respondError(c, err)
		return
	}
	result := gin.H{
		"question":    qaQuestionResponse(q, isStaff || q.CreatedBy == reader),
		"attachments": qaAttachmentResponses(qAtts),
	}
	if answer != nil {
		result["answer"] = QAAnswerResponse{
			ID: answer.ID, Body: answer.Body, AuthorName: answer.AuthorName,
			CreatedAt: answer.CreatedAt, UpdatedAt: answer.UpdatedAt,
		}
		result["answer_attachments"] = qaAttachmentResponses(aAtts)
	}
	if rejection != nil {
		result["rejection"] = gin.H{"reason": rejection.Reason, "rejected_at": rejection.RejectedAt}
	}
	response.OK(c, result)
}

func qaAttachmentResponses(atts []classroom.QAAttachment) []AttachmentResponse {
	result := make([]AttachmentResponse, len(atts))
	for i, a := range atts {
		result[i] = AttachmentResponse{ID: a.ID, DisplayName: a.DisplayName, OrderIndex: a.OrderIndex}
	}
	return result
}

// ListQAQuestions serves the answered board by default; ?status=pending
// is the teacher's queue (students get their own), ?status=rejected is
// always scoped to the caller for non-staff, ?faq=true narrows to FAQs.
func (h *Handler) ListQAQuestions(c *gin.Context) {
	filter := classroom.QAFilter{}
	if v := c.Query("status"); v != "" {
		filter.Status = classroom.QAStatus(v)
	}
	if v := c.Query("faq"); v != "" {
		faq := v == "true"
		filter.FAQ = &faq
	}
	isStaff := teaching(c)
	if filter.Status == classroom.QAPending || filter.Status == classroom.QARejected {
		if !isStaff {
			me := middleware.GetUserID(c)
			filter.Mine = &me
		}
	}
	questions, err := h.qa.List(c.Request.Context(), offeringID(c), filter)
	if err != nil {
		h.respondError(c, err)
		return
	}
	reader := middleware.GetUserID(c)
	result := make([]QAQuestionResponse, len(questions))
	for i := range questions {
		result[i] = qaQuestionResponse(&questions[i], isStaff || questions[i].CreatedBy == reader)
	}
	response.OK(c, result)
}

func (h *Handler) UpdateQAQuestion(c *gin.Context) {
	var req struct {
		Title *string `json:"title" binding:"omitempty,max=255"`
		Body  *string `json:"body" binding:"omitempty,max=10000"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	q, err := h.qa.Update(c.Request.Context(), offeringID(c), targetID(c),
		middleware.GetUserID(c), teaching(c), req.Title, req.Body)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, gin.H{"id": q.ID, "status": q.Status, "version": q.Version})
}

func (h *Handler) DeleteQAQuestion(c *gin.Context) {
	if err := h.qa.Delete(c.Request.Context(), offeringID(c), targetID(c),
		middleware.GetUserID(c), teaching(c)); err != nil {
		h.respondError(c, err)
		return
	}
	response.NoContent(c)
}

// QACustom dispatches the question's colon methods.
func (h *Handler) QACustom(c *gin.Context) {
	ctx := c.Request.Context()
	switch customAction(c) {
	case "answer":
		var req struct {
			Body         string           `json:"body" binding:"required,max=10000"`
			QuestionEdit *string          `json:"question_edit" binding:"omitempty,max=10000"`
			Files        []FileRefRequest `json:"files" binding:"omitempty,max=10,dive"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		answer, err := h.qa.Answer(ctx, offeringID(c), targetID(c), middleware.GetUserID(c), classroom.AnswerInput{
			Body: req.Body, QuestionEdit: req.QuestionEdit, Files: fileRefs(req.Files),
		})
		if err != nil {
			h.respondError(c, err)
			return
		}
		response.OK(c, gin.H{"id": answer.ID})
	case "reject":
		var req struct {
			Reason string `json:"reason" binding:"required,max=2000"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		if err := h.qa.Reject(ctx, offeringID(c), targetID(c), middleware.GetUserID(c), req.Reason); err != nil {
			h.respondError(c, err)
			return
		}
		response.OK(c, gin.H{"status": "rejected"})
	default:
		response.NotFound(c, "unknown method")
	}
}

func (h *Handler) DownloadQAAttachment(c *gin.Context) {
	attachmentID, err := uuid.Parse(c.Param("attachmentId"))
	if err != nil {
		response.NotFound(c, "attachment not found")
		return
	}
	url, err := h.qa.PresignAttachment(c.Request.Context(), offeringID(c), targetID(c), attachmentID,
		middleware.GetUserID(c), teaching(c))
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.Redirect(http.StatusTemporaryRedirect, url)
}
