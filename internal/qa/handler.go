package qa

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ranjdotdev/e-campus-server/internal/authz"
	"github.com/ranjdotdev/e-campus-server/internal/middleware"
	"github.com/ranjdotdev/e-campus-server/internal/pagination"
	"github.com/ranjdotdev/e-campus-server/internal/response"
	"go.uber.org/zap"
)

type Handler struct {
	service *Service
	log     *zap.Logger
}

func NewHandler(service *Service, log *zap.Logger) *Handler {
	return &Handler{
		service: service,
		log:     log,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, auth gin.HandlerFunc) {
	offerings := rg.Group("/offerings/:id")
	offerings.Use(auth)
	{
		offerings.GET("/questions", h.ListQuestions)
		offerings.GET("/questions/pending", h.ListPendingQuestions)
		offerings.GET("/questions/faq", h.ListFAQ)
		offerings.POST("/questions", h.AskQuestion)
		offerings.POST("/questions/faq", h.CreateFAQ)
	}

	qa := rg.Group("/qa")
	qa.Use(auth)
	{
		qa.GET("/:id", h.GetQuestion)
		qa.PUT("/:id", h.UpdateQuestion)
		qa.DELETE("/:id", h.DeleteQuestion)
		qa.POST("/:id/answer", h.AnswerQuestion)
		qa.PUT("/:id/answer", h.UpdateAnswer)
		qa.POST("/:id/reject", h.RejectQuestion)
	}
}

func (h *Handler) AskQuestion(c *gin.Context) {
	offeringID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid offering id")
		return
	}

	var req AskQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	userID := middleware.GetUserID(c)

	if !authz.Check(c, authz.ResourceQA, authz.ActionCreate, offeringID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	q, err := h.service.AskQuestion(c.Request.Context(), offeringID, userID, req.Title, req.Body, req.IsAnonymous)
	if err != nil {
		h.handleError(c, err)
		return
	}

	resp := QuestionResponse{
		ID:          q.ID,
		OfferingID:  q.OfferingID,
		Title:       q.Title,
		Body:        q.Body,
		IsAnonymous: q.IsAnonymous,
		Status:      q.Status,
		AuthorID:    &q.CreatedBy,
		CreatedAt:   q.CreatedAt,
	}
	response.Created(c, resp)
}

func (h *Handler) CreateFAQ(c *gin.Context) {
	offeringID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid offering id")
		return
	}

	var req CreateFAQRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	if !authz.Check(c, authz.ResourceQA, authz.ActionUpdate, offeringID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	teacherID := middleware.GetUserID(c)

	q, a, err := h.service.CreateFAQ(c.Request.Context(), offeringID, teacherID, req.Title, req.QuestionBody, req.AnswerBody)
	if err != nil {
		h.handleError(c, err)
		return
	}

	resp := QuestionResponse{
		ID:         q.ID,
		OfferingID: q.OfferingID,
		Title:      q.Title,
		Body:       q.Body,
		IsFAQ:      true,
		Status:     StatusAnswered,
		AuthorID:   &q.CreatedBy,
		CreatedAt:  q.CreatedAt,
		Answer: &AnswerResponse{
			ID:        a.ID,
			Body:      a.Body,
			AuthorID:  a.CreatedBy,
			CreatedAt: a.CreatedAt,
		},
	}
	response.Created(c, resp)
}

func (h *Handler) GetQuestion(c *gin.Context) {
	questionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid question id")
		return
	}

	q, answer, qAttachments, aAttachments, err := h.service.GetQuestion(c.Request.Context(), questionID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	if !authz.Check(c, authz.ResourceQA, authz.ActionGet, q.OfferingID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	userID := middleware.GetUserID(c)
	isTeacher := isOfferingStaff(c, q.OfferingID)

	if !CanViewQuestion(&q.Question, userID, isTeacher) {
		response.NotFound(c, "question not found")
		return
	}

	var rejection *QuestionRejectionWithUser
	if q.Status == StatusRejected {
		rejection, _ = h.service.GetRejection(c.Request.Context(), questionID)
	}

	resp := ToQuestionResponse(q, answer, rejection, qAttachments, aAttachments, isTeacher)
	response.OK(c, resp)
}

func (h *Handler) ListQuestions(c *gin.Context) {
	offeringID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid offering id")
		return
	}

	if !authz.Check(c, authz.ResourceQA, authz.ActionGet, offeringID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	isTeacher := isOfferingStaff(c, offeringID)
	params := pagination.ParsePageParams(c)

	questions, hasMore, err := h.service.ListQuestions(c.Request.Context(), offeringID, nil, params)
	if err != nil {
		if errors.Is(err, pagination.ErrInvalidCursor) {
			response.BadRequest(c, "invalid cursor")
			return
		}
		h.log.Error("list questions failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	result := pagination.PageResult[QuestionResponse]{
		Data:    ToQuestionListResponses(questions, isTeacher),
		HasMore: hasMore,
	}
	if hasMore && len(questions) > 0 {
		last := questions[len(questions)-1]
		result.NextCursor = pagination.EncodeCursor(last.CreatedAt, last.ID)
	}

	response.OK(c, result)
}

func (h *Handler) ListFAQ(c *gin.Context) {
	offeringID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid offering id")
		return
	}

	if !authz.Check(c, authz.ResourceQA, authz.ActionGet, offeringID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	isTeacher := isOfferingStaff(c, offeringID)
	params := pagination.ParsePageParams(c)

	isFAQ := true
	questions, hasMore, err := h.service.ListQuestions(c.Request.Context(), offeringID, &isFAQ, params)
	if err != nil {
		if errors.Is(err, pagination.ErrInvalidCursor) {
			response.BadRequest(c, "invalid cursor")
			return
		}
		h.log.Error("list FAQ failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	result := pagination.PageResult[QuestionResponse]{
		Data:    ToQuestionListResponses(questions, isTeacher),
		HasMore: hasMore,
	}
	if hasMore && len(questions) > 0 {
		last := questions[len(questions)-1]
		result.NextCursor = pagination.EncodeCursor(last.CreatedAt, last.ID)
	}

	response.OK(c, result)
}

func (h *Handler) ListPendingQuestions(c *gin.Context) {
	offeringID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid offering id")
		return
	}

	if !authz.Check(c, authz.ResourceQA, authz.ActionUpdate, offeringID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	params := pagination.ParsePageParams(c)

	questions, hasMore, err := h.service.ListPendingQuestions(c.Request.Context(), offeringID, params)
	if err != nil {
		if errors.Is(err, pagination.ErrInvalidCursor) {
			response.BadRequest(c, "invalid cursor")
			return
		}
		h.log.Error("list pending questions failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	result := pagination.PageResult[QuestionResponse]{
		Data:    ToQuestionListResponses(questions, true),
		HasMore: hasMore,
	}
	if hasMore && len(questions) > 0 {
		last := questions[len(questions)-1]
		result.NextCursor = pagination.EncodeCursor(last.CreatedAt, last.ID)
	}

	response.OK(c, result)
}

func (h *Handler) UpdateQuestion(c *gin.Context) {
	questionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid question id")
		return
	}

	var req UpdateQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	q, err := h.service.GetQuestionByID(c.Request.Context(), questionID)
	if err != nil {
		h.log.Error("get question failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if q == nil || q.DeletedAt != nil {
		response.NotFound(c, "question not found")
		return
	}

	if !authz.Check(c, authz.ResourceQA, authz.ActionUpdate, q.OfferingID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	userID := middleware.GetUserID(c)
	isTeacher := isOfferingStaff(c, q.OfferingID)

	updated, err := h.service.UpdateQuestion(c.Request.Context(), questionID, userID, isTeacher, req.Title, req.Body)
	if err != nil {
		h.handleError(c, err)
		return
	}

	resp := QuestionResponse{
		ID:         updated.ID,
		OfferingID: updated.OfferingID,
		Title:      updated.Title,
		Body:       updated.Body,
		Status:     updated.Status,
		EditedBy:   updated.EditedBy,
		CreatedAt:  updated.CreatedAt,
		UpdatedAt:  updated.UpdatedAt,
	}
	response.OK(c, resp)
}

func (h *Handler) DeleteQuestion(c *gin.Context) {
	questionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid question id")
		return
	}

	q, err := h.service.GetQuestionByID(c.Request.Context(), questionID)
	if err != nil {
		h.log.Error("get question failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if q == nil || q.DeletedAt != nil {
		response.NotFound(c, "question not found")
		return
	}

	// authz.Check gates course membership; service.DeleteQuestion enforces author-only
	// deletion of pending questions. Teachers moderate via Reject, not Delete.
	if !authz.Check(c, authz.ResourceQA, authz.ActionDelete, q.OfferingID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	userID := middleware.GetUserID(c)

	if err := h.service.DeleteQuestion(c.Request.Context(), questionID, userID); err != nil {
		h.handleError(c, err)
		return
	}

	response.NoContent(c)
}

func (h *Handler) AnswerQuestion(c *gin.Context) {
	questionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid question id")
		return
	}

	var req AnswerQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	q, err := h.service.GetQuestionByID(c.Request.Context(), questionID)
	if err != nil {
		h.log.Error("get question failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if q == nil || q.DeletedAt != nil {
		response.NotFound(c, "question not found")
		return
	}

	if !authz.Check(c, authz.ResourceQA, authz.ActionUpdate, q.OfferingID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	teacherID := middleware.GetUserID(c)

	updatedQ, a, err := h.service.AnswerQuestion(c.Request.Context(), questionID, teacherID, req.Body, req.QuestionEdit)
	if err != nil {
		h.handleError(c, err)
		return
	}

	resp := QuestionResponse{
		ID:         updatedQ.ID,
		OfferingID: updatedQ.OfferingID,
		Title:      updatedQ.Title,
		Body:       updatedQ.Body,
		Status:     StatusAnswered,
		EditedBy:   updatedQ.EditedBy,
		CreatedAt:  updatedQ.CreatedAt,
		UpdatedAt:  updatedQ.UpdatedAt,
		Answer: &AnswerResponse{
			ID:        a.ID,
			Body:      a.Body,
			AuthorID:  a.CreatedBy,
			CreatedAt: a.CreatedAt,
		},
	}
	response.OK(c, resp)
}

func (h *Handler) UpdateAnswer(c *gin.Context) {
	questionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid question id")
		return
	}

	var req UpdateAnswerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	q, err := h.service.GetQuestionByID(c.Request.Context(), questionID)
	if err != nil {
		h.log.Error("get question failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if q == nil || q.DeletedAt != nil {
		response.NotFound(c, "question not found")
		return
	}

	if !authz.Check(c, authz.ResourceQA, authz.ActionUpdate, q.OfferingID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	teacherID := middleware.GetUserID(c)

	a, err := h.service.UpdateAnswer(c.Request.Context(), questionID, teacherID, req.Body)
	if err != nil {
		h.handleError(c, err)
		return
	}

	resp := AnswerResponse{
		ID:        a.ID,
		Body:      a.Body,
		AuthorID:  a.CreatedBy,
		CreatedAt: a.CreatedAt,
		UpdatedAt: a.UpdatedAt,
	}
	response.OK(c, resp)
}

func (h *Handler) RejectQuestion(c *gin.Context) {
	questionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid question id")
		return
	}

	var req RejectQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	q, err := h.service.GetQuestionByID(c.Request.Context(), questionID)
	if err != nil {
		h.log.Error("get question failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if q == nil || q.DeletedAt != nil {
		response.NotFound(c, "question not found")
		return
	}

	if !authz.Check(c, authz.ResourceQA, authz.ActionUpdate, q.OfferingID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	teacherID := middleware.GetUserID(c)

	if err := h.service.RejectQuestion(c.Request.Context(), questionID, teacherID, req.Reason); err != nil {
		h.handleError(c, err)
		return
	}

	response.NoContent(c)
}

func isOfferingStaff(c *gin.Context, offeringID uuid.UUID) bool {
	role := authz.CourseRole(c, offeringID)
	return role == "teacher" || role == "assistant"
}

func (h *Handler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrQuestionNotFound):
		response.NotFound(c, "question not found")
	case errors.Is(err, ErrAnswerNotFound):
		response.NotFound(c, "answer not found")
	case errors.Is(err, ErrOfferingNotFound):
		response.NotFound(c, "offering not found")
	case errors.Is(err, ErrNotAuthorized):
		response.Forbidden(c, "not authorized")
	case errors.Is(err, ErrNotAuthor):
		response.Forbidden(c, "not question author")
	case errors.Is(err, ErrUserMuted):
		response.Forbidden(c, "user is muted")
	case errors.Is(err, ErrQuestionRejected):
		response.BadRequest(c, "question was rejected")
	case errors.Is(err, ErrNotPending):
		response.BadRequest(c, "question is not pending")
	case errors.Is(err, ErrEmptyTitle):
		response.BadRequest(c, "title required")
	case errors.Is(err, ErrEmptyBody):
		response.BadRequest(c, "body required")
	case errors.Is(err, ErrTitleTooLong):
		response.BadRequest(c, "title too long")
	case errors.Is(err, ErrEmptyReason):
		response.BadRequest(c, "rejection reason required")
	default:
		h.log.Error("handler error", zap.Error(err))
		response.InternalError(c)
	}
}
