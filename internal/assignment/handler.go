package assignment

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/randdotdev/e-campus-server/internal/authz"
	"github.com/randdotdev/e-campus-server/internal/middleware"
	"github.com/randdotdev/e-campus-server/internal/response"
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

func (h *Handler) CreateAssignment(c *gin.Context) {
	offeringID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid offering id")
		return
	}

	if !authz.Check(c, authz.ResourceAssignment, authz.ActionCreate, offeringID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	var req CreateAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	userID := middleware.GetUserID(c)

	a := &Assignment{
		OfferingID: offeringID,
		Title:      req.Title,
		Body:       req.Body,
		Type:       req.Type,
		Deadline:   req.Deadline,
		MaxScore:   req.MaxScore,
		AllowLate:  req.AllowLate,
		PublishAt:  req.PublishAt,
		CreatedBy:  &userID,
	}

	if err := h.service.CreateAssignment(c.Request.Context(), a); err != nil {
		h.log.Error("create assignment failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.Created(c, ToAssignmentResponse(a, time.Now()))
}

func (h *Handler) GetAssignment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid assignment id")
		return
	}

	a, err := h.service.GetAssignment(c.Request.Context(), id)
	if err != nil {
		h.log.Error("get assignment failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if a == nil {
		response.NotFound(c, "assignment not found")
		return
	}

	now := time.Now()

	if !authz.Check(c, authz.ResourceAssignment, authz.ActionGet, a.OfferingID) {
		response.Forbidden(c, "not enrolled")
		return
	}
	isStaff := isOfferingStaff(c, a.OfferingID)
	if !isStaff && !IsPublished(a.PublishAt, now) {
		response.NotFound(c, "assignment not found")
		return
	}

	attachments, err := h.service.GetAttachments(c.Request.Context(), id)
	if err != nil {
		h.log.Error("get attachments failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, ToAssignmentWithAttachmentsResponse(a, attachments, now))
}

func (h *Handler) ListAssignments(c *gin.Context) {
	offeringID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid offering id")
		return
	}

	if !authz.Check(c, authz.ResourceAssignment, authz.ActionGet, offeringID) {
		response.Forbidden(c, "not enrolled")
		return
	}
	isStaff := isOfferingStaff(c, offeringID)

	var assignments []Assignment
	if isStaff {
		assignments, err = h.service.ListAssignments(c.Request.Context(), offeringID)
	} else {
		assignments, err = h.service.ListPublishedAssignments(c.Request.Context(), offeringID)
	}

	if err != nil {
		h.log.Error("list assignments failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, ToAssignmentsResponse(assignments, time.Now()))
}

func (h *Handler) UpdateAssignment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid assignment id")
		return
	}

	a, err := h.service.GetAssignment(c.Request.Context(), id)
	if err != nil {
		h.log.Error("get assignment failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if a == nil {
		response.NotFound(c, "assignment not found")
		return
	}

	if !authz.Check(c, authz.ResourceAssignment, authz.ActionUpdate, a.OfferingID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	var req UpdateAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	updated, err := h.service.UpdateAssignment(c.Request.Context(), id, ToAssignmentUpdates(req))
	if errors.Is(err, ErrAssignmentNotFound) {
		response.NotFound(c, "assignment not found")
	} else if err != nil {
		h.log.Error("update assignment failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.OK(c, ToAssignmentResponse(updated, time.Now()))
	}
}

func (h *Handler) DeleteAssignment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid assignment id")
		return
	}

	a, err := h.service.GetAssignment(c.Request.Context(), id)
	if err != nil {
		h.log.Error("get assignment failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if a == nil {
		response.NotFound(c, "assignment not found")
		return
	}

	if !authz.Check(c, authz.ResourceAssignment, authz.ActionDelete, a.OfferingID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	if err := h.service.DeleteAssignment(c.Request.Context(), id); err != nil {
		h.log.Error("delete assignment failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.NoContent(c)
	}
}

func (h *Handler) PublishScores(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid assignment id")
		return
	}

	a, err := h.service.GetAssignment(c.Request.Context(), id)
	if err != nil {
		h.log.Error("get assignment failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if a == nil {
		response.NotFound(c, "assignment not found")
		return
	}

	if !authz.Check(c, authz.ResourceAssignment, authz.ActionUpdate, a.OfferingID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	if err := h.service.PublishScores(c.Request.Context(), id); err != nil {
		h.log.Error("publish scores failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.OK(c, gin.H{"scores_public": true})
	}
}

func (h *Handler) AddAttachment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid assignment id")
		return
	}

	a, err := h.service.GetAssignment(c.Request.Context(), id)
	if err != nil {
		h.log.Error("get assignment failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if a == nil {
		response.NotFound(c, "assignment not found")
		return
	}

	if !authz.Check(c, authz.ResourceAssignment, authz.ActionUpdate, a.OfferingID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	userID := middleware.GetUserID(c)

	var req AddAttachmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	att := &AssignmentAttachment{
		AssignmentID: id,
		StoredFileID: req.StoredFileID,
		DisplayName:  req.DisplayName,
		OrderIndex:   req.OrderIndex,
		AddedBy:      &userID,
	}

	if err := h.service.AddAttachment(c.Request.Context(), att); err != nil {
		h.log.Error("add attachment failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.Created(c, ToAttachmentResponse(att))
}

func (h *Handler) DeleteAttachment(c *gin.Context) {
	assignmentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid assignment id")
		return
	}

	attachmentID, err := uuid.Parse(c.Param("attachment_id"))
	if err != nil {
		response.BadRequest(c, "invalid attachment id")
		return
	}

	a, err := h.service.GetAssignment(c.Request.Context(), assignmentID)
	if err != nil {
		h.log.Error("get assignment failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if a == nil {
		response.NotFound(c, "assignment not found")
		return
	}

	if !authz.Check(c, authz.ResourceAssignment, authz.ActionUpdate, a.OfferingID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	if err := h.service.DeleteAttachment(c.Request.Context(), attachmentID); err != nil {
		if errors.Is(err, ErrAttachmentNotFound) {
			response.NotFound(c, "attachment not found")
		} else {
			h.log.Error("delete attachment failed", zap.Error(err))
			response.InternalError(c)
		}
	} else {
		response.NoContent(c)
	}
}

func (h *Handler) CreateSubmission(c *gin.Context) {
	assignmentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid assignment id")
		return
	}

	var req CreateSubmissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	userID := middleware.GetUserID(c)

	sub, err := h.service.CreateSubmission(c.Request.Context(), assignmentID, userID, req.Content, ToFileInputs(req.Files))
	if errors.Is(err, ErrAssignmentNotFound) {
		response.NotFound(c, "assignment not found")
	} else if errors.Is(err, ErrNotPublished) {
		response.BadRequest(c, "assignment not published")
	} else if errors.Is(err, ErrNotEnrolled) {
		response.Forbidden(c, "not enrolled")
	} else if errors.Is(err, ErrSubmissionExists) {
		response.Conflict(c, "submission already exists")
	} else if errors.Is(err, ErrFileNotOwned) {
		response.BadRequest(c, "file not owned by student")
	} else if err != nil {
		h.log.Error("create submission failed", zap.Error(err))
		response.InternalError(c)
	} else {
		a, _ := h.service.GetAssignment(c.Request.Context(), assignmentID)
		files, _ := h.service.GetSubmissionFiles(c.Request.Context(), sub.ID)
		response.Created(c, ToSubmissionResponse(sub, files, a.Deadline))
	}
}

func (h *Handler) GetMySubmission(c *gin.Context) {
	assignmentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid assignment id")
		return
	}

	userID := middleware.GetUserID(c)

	sub, err := h.service.GetMySubmission(c.Request.Context(), assignmentID, userID)
	if errors.Is(err, ErrSubmissionNotFound) {
		response.NotFound(c, "submission not found")
		return
	} else if err != nil {
		h.log.Error("get submission failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	a, _ := h.service.GetAssignment(c.Request.Context(), assignmentID)
	files, _ := h.service.GetSubmissionFiles(c.Request.Context(), sub.ID)

	response.OK(c, ToSubmissionWithScoreResponse(sub, files, a.Deadline, a.ScoresPublic))
}

func (h *Handler) GetSubmission(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid submission id")
		return
	}

	sub, err := h.service.GetSubmission(c.Request.Context(), id)
	if err != nil {
		h.log.Error("get submission failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if sub == nil {
		response.NotFound(c, "submission not found")
		return
	}

	a, err := h.service.GetAssignment(c.Request.Context(), sub.AssignmentID)
	if err != nil {
		h.log.Error("get assignment failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	// ActionUpdate intentionally used: this is a staff-only view of a student's submission.
	// Students access their own via GetMySubmission; ActionGet would expose all submissions to students.
	if !authz.Check(c, authz.ResourceAssignment, authz.ActionUpdate, a.OfferingID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	files, _ := h.service.GetSubmissionFiles(c.Request.Context(), sub.ID)

	resp := SubmissionTeacherResponse{
		SubmissionResponse: ToSubmissionResponse(sub, files, a.Deadline),
		Score:              sub.Score,
		Feedback:           sub.Feedback,
		GradedBy:           sub.GradedBy,
		GradedAt:           sub.GradedAt,
	}

	response.OK(c, resp)
}

func (h *Handler) ListSubmissions(c *gin.Context) {
	assignmentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid assignment id")
		return
	}

	a, err := h.service.GetAssignment(c.Request.Context(), assignmentID)
	if err != nil {
		h.log.Error("get assignment failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if a == nil {
		response.NotFound(c, "assignment not found")
		return
	}

	// ActionUpdate intentionally used: staff-only listing of all student submissions.
	// ActionGet would allow enrolled students to list each other's submissions.
	if !authz.Check(c, authz.ResourceAssignment, authz.ActionUpdate, a.OfferingID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	submissions, err := h.service.ListSubmissions(c.Request.Context(), assignmentID)
	if err != nil {
		h.log.Error("list submissions failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	result := make([]SubmissionTeacherResponse, len(submissions))
	for i := range submissions {
		files, _ := h.service.GetSubmissionFiles(c.Request.Context(), submissions[i].ID)
		result[i] = ToSubmissionTeacherResponse(&submissions[i], files, a.Deadline)
	}

	response.OK(c, result)
}

func (h *Handler) UpdateSubmission(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid submission id")
		return
	}

	var req UpdateSubmissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	userID := middleware.GetUserID(c)

	sub, err := h.service.UpdateSubmission(c.Request.Context(), id, userID, req.Content, ToFileInputs(req.Files))
	if errors.Is(err, ErrSubmissionNotFound) {
		response.NotFound(c, "submission not found")
	} else if errors.Is(err, ErrCannotModify) {
		response.Forbidden(c, "cannot modify submission")
	} else if errors.Is(err, ErrFileNotOwned) {
		response.BadRequest(c, "file not owned by student")
	} else if err != nil {
		h.log.Error("update submission failed", zap.Error(err))
		response.InternalError(c)
	} else {
		a, _ := h.service.GetAssignment(c.Request.Context(), sub.AssignmentID)
		files, _ := h.service.GetSubmissionFiles(c.Request.Context(), sub.ID)
		response.OK(c, ToSubmissionResponse(sub, files, a.Deadline))
	}
}

func (h *Handler) SubmitSubmission(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid submission id")
		return
	}

	userID := middleware.GetUserID(c)

	sub, err := h.service.SubmitSubmission(c.Request.Context(), id, userID)
	if errors.Is(err, ErrSubmissionNotFound) {
		response.NotFound(c, "submission not found")
	} else if errors.Is(err, ErrAlreadySubmitted) {
		response.Conflict(c, "already submitted")
	} else if errors.Is(err, ErrSubmissionsClosed) {
		response.BadRequest(c, "submissions closed")
	} else if errors.Is(err, ErrNoContent) {
		response.BadRequest(c, "submission must have content or files")
	} else if err != nil {
		h.log.Error("submit submission failed", zap.Error(err))
		response.InternalError(c)
	} else {
		a, _ := h.service.GetAssignment(c.Request.Context(), sub.AssignmentID)
		files, _ := h.service.GetSubmissionFiles(c.Request.Context(), sub.ID)
		response.OK(c, ToSubmissionResponse(sub, files, a.Deadline))
	}
}

func (h *Handler) DeleteSubmission(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid submission id")
		return
	}

	userID := middleware.GetUserID(c)

	err = h.service.DeleteSubmission(c.Request.Context(), id, userID)
	if errors.Is(err, ErrSubmissionNotFound) {
		response.NotFound(c, "submission not found")
	} else if errors.Is(err, ErrNotDraft) {
		response.Forbidden(c, "can only delete draft submissions")
	} else if err != nil {
		h.log.Error("delete submission failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.NoContent(c)
	}
}

func (h *Handler) GradeSubmission(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid submission id")
		return
	}

	sub, err := h.service.GetSubmission(c.Request.Context(), id)
	if err != nil {
		h.log.Error("get submission failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if sub == nil {
		response.NotFound(c, "submission not found")
		return
	}

	a, err := h.service.GetAssignment(c.Request.Context(), sub.AssignmentID)
	if err != nil {
		h.log.Error("get assignment failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	if !authz.Check(c, authz.ResourceAssignment, authz.ActionUpdate, a.OfferingID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	userID := middleware.GetUserID(c)

	var req GradeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	graded, err := h.service.GradeSubmission(c.Request.Context(), id, userID, req.Score, req.Feedback)
	if errors.Is(err, ErrInvalidScore) {
		response.BadRequest(c, "score must be between 0 and max score")
	} else if err != nil {
		h.log.Error("grade submission failed", zap.Error(err))
		response.InternalError(c)
	} else {
		files, _ := h.service.GetSubmissionFiles(c.Request.Context(), graded.ID)
		resp := SubmissionTeacherResponse{
			SubmissionResponse: ToSubmissionResponse(graded, files, a.Deadline),
			Score:              graded.Score,
			Feedback:           graded.Feedback,
			GradedBy:           graded.GradedBy,
			GradedAt:           graded.GradedAt,
		}
		response.OK(c, resp)
	}
}

func isOfferingStaff(c *gin.Context, offeringID uuid.UUID) bool {
	role := authz.CourseRole(c, offeringID)
	return role == "teacher" || role == "assistant"
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	assignments := r.Group("/assignments")
	assignments.Use(authMiddleware)
	{
		assignments.GET("/:id", h.GetAssignment)
		assignments.PUT("/:id", h.UpdateAssignment)
		assignments.DELETE("/:id", h.DeleteAssignment)
		assignments.PUT("/:id/publish-scores", h.PublishScores)
		assignments.POST("/:id/attachments", h.AddAttachment)
		assignments.DELETE("/:id/attachments/:attachment_id", h.DeleteAttachment)
		assignments.GET("/:id/submissions", h.ListSubmissions)
		assignments.POST("/:id/submissions", h.CreateSubmission)
		assignments.GET("/:id/submissions/me", h.GetMySubmission)
	}

	submissions := r.Group("/submissions")
	submissions.Use(authMiddleware)
	{
		submissions.GET("/:id", h.GetSubmission)
		submissions.PUT("/:id", h.UpdateSubmission)
		submissions.POST("/:id/submit", h.SubmitSubmission)
		submissions.DELETE("/:id", h.DeleteSubmission)
		submissions.PUT("/:id/grade", h.GradeSubmission)
	}

	offerings := r.Group("/offerings")
	offerings.Use(authMiddleware)
	{
		offerings.GET("/:id/assignments", h.ListAssignments)
		offerings.POST("/:id/assignments", h.CreateAssignment)
	}
}
