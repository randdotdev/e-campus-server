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

type AssignmentResponse struct {
	ID           uuid.UUID  `json:"id"`
	OfferingID   uuid.UUID  `json:"offering_id"`
	Title        string     `json:"title"`
	Body         *string    `json:"body"`
	Type         *string    `json:"type"`
	Deadline     time.Time  `json:"deadline"`
	MaxScore     float64    `json:"max_score"`
	AllowLate    bool       `json:"allow_late"`
	PublishAt    *time.Time `json:"publish_at"`
	ScoresPublic bool       `json:"scores_public"`
	Version      int64      `json:"version"`
	CreatedAt    time.Time  `json:"created_at"`
}

func assignmentResponse(a *classroom.Assignment) AssignmentResponse {
	var aType *string
	if a.Type != nil {
		t := string(*a.Type)
		aType = &t
	}
	return AssignmentResponse{
		ID: a.ID, OfferingID: a.OfferingID, Title: a.Title, Body: a.Body,
		Type: aType, Deadline: a.Deadline, MaxScore: a.MaxScore,
		AllowLate: a.AllowLate, PublishAt: a.PublishAt, ScoresPublic: a.ScoresPublic,
		Version: a.Version, CreatedAt: a.CreatedAt,
	}
}

type SubmissionResponse struct {
	ID           uuid.UUID  `json:"id"`
	AssignmentID uuid.UUID  `json:"assignment_id"`
	StudentID    uuid.UUID  `json:"student_id"`
	Content      *string    `json:"content"`
	SubmittedAt  *time.Time `json:"submitted_at"`
	Score        *float64   `json:"score"`
	Feedback     *string    `json:"feedback"`
	GradedAt     *time.Time `json:"graded_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at"`
}

func submissionResponse(s *classroom.Submission) SubmissionResponse {
	return SubmissionResponse{
		ID: s.ID, AssignmentID: s.AssignmentID, StudentID: s.StudentID,
		Content: s.Content, SubmittedAt: s.SubmittedAt, Score: s.Score,
		Feedback: s.Feedback, GradedAt: s.GradedAt,
		CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}
}

type SubmissionWithStudentResponse struct {
	SubmissionResponse
	StudentName   string  `json:"student_name"`
	StudentEmail  string  `json:"student_email"`
	StudentAvatar *string `json:"student_avatar"`
}

type CreateAssignmentRequest struct {
	Title     string     `json:"title" binding:"required,max=255"`
	Body      *string    `json:"body"`
	Type      *string    `json:"type" binding:"omitempty,oneof=theory practice"`
	Deadline  time.Time  `json:"deadline" binding:"required"`
	MaxScore  float64    `json:"max_score" binding:"required,gt=0"`
	AllowLate bool       `json:"allow_late"`
	PublishAt *time.Time `json:"publish_at"`
}

type UpdateAssignmentRequest struct {
	Title        *string    `json:"title" binding:"omitempty,max=255"`
	Body         *string    `json:"body"`
	Type         *string    `json:"type" binding:"omitempty,oneof=theory practice"`
	Deadline     *time.Time `json:"deadline"`
	MaxScore     *float64   `json:"max_score" binding:"omitempty,gt=0"`
	AllowLate    *bool      `json:"allow_late"`
	PublishAt    *time.Time `json:"publish_at"`
	ScoresPublic *bool      `json:"scores_public"`
}

// FileRefRequest is one drive-file reference in a submission body.
type FileRefRequest struct {
	UploadID    uuid.UUID `json:"upload_id" binding:"required"`
	DisplayName string    `json:"display_name" binding:"omitempty,max=255"`
}

type SaveDraftRequest struct {
	Content *string          `json:"content"`
	Files   []FileRefRequest `json:"files" binding:"omitempty,max=20,dive"`
}

type GradeSubmissionRequest struct {
	StudentID uuid.UUID `json:"student_id" binding:"required"`
	Score     float64   `json:"score"`
	Feedback  *string   `json:"feedback"`
}

func fileRefs(reqs []FileRefRequest) []classroom.FileRef {
	refs := make([]classroom.FileRef, len(reqs))
	for i, r := range reqs {
		refs[i] = classroom.FileRef{UploadID: r.UploadID, DisplayName: r.DisplayName}
	}
	return refs
}

func sessionType(s *string) *classroom.SessionType {
	if s == nil {
		return nil
	}
	t := classroom.SessionType(*s)
	return &t
}

func (h *Handler) CreateAssignment(c *gin.Context) {
	var req CreateAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	a, err := h.assignments.Create(c.Request.Context(), classroom.CreateAssignmentInput{
		OfferingID: offeringID(c), CreatedBy: middleware.GetUserID(c),
		Title: req.Title, Body: req.Body, Type: sessionType(req.Type),
		Deadline: req.Deadline, MaxScore: req.MaxScore,
		AllowLate: req.AllowLate, PublishAt: req.PublishAt,
	})
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Created(c, assignmentResponse(a))
}

func (h *Handler) GetAssignment(c *gin.Context) {
	a, attachments, err := h.assignments.Get(c.Request.Context(), offeringID(c), targetID(c), studentView(c))
	if err != nil {
		h.respondError(c, err)
		return
	}
	atts := make([]AttachmentResponse, len(attachments))
	for i, att := range attachments {
		atts[i] = AttachmentResponse{ID: att.ID, DisplayName: att.DisplayName, OrderIndex: att.OrderIndex}
	}
	response.OK(c, gin.H{"assignment": assignmentResponse(a), "attachments": atts})
}

func (h *Handler) ListAssignments(c *gin.Context) {
	assignments, err := h.assignments.List(c.Request.Context(), offeringID(c), studentView(c))
	if err != nil {
		h.respondError(c, err)
		return
	}
	result := make([]AssignmentResponse, len(assignments))
	for i := range assignments {
		result[i] = assignmentResponse(&assignments[i])
	}
	response.OK(c, result)
}

func (h *Handler) UpdateAssignment(c *gin.Context) {
	var req UpdateAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	a, err := h.assignments.Update(c.Request.Context(), offeringID(c), targetID(c), classroom.UpdateAssignmentInput{
		Title: req.Title, Body: req.Body, Type: sessionType(req.Type),
		Deadline: req.Deadline, MaxScore: req.MaxScore, AllowLate: req.AllowLate,
		PublishAt: req.PublishAt, ScoresPublic: req.ScoresPublic,
	})
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, assignmentResponse(a))
}

func (h *Handler) DeleteAssignment(c *gin.Context) {
	if err := h.assignments.Delete(c.Request.Context(), offeringID(c), targetID(c)); err != nil {
		h.respondError(c, err)
		return
	}
	response.NoContent(c)
}

// AssignmentCustom dispatches the assignment's colon methods.
func (h *Handler) AssignmentCustom(c *gin.Context) {
	ctx := c.Request.Context()
	switch customAction(c) {
	case "attach":
		var req AttachRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		att, err := h.assignments.Attach(ctx, offeringID(c), targetID(c), middleware.GetUserID(c),
			classroom.FileRef{UploadID: req.UploadID, DisplayName: req.DisplayName})
		if err != nil {
			h.respondError(c, err)
			return
		}
		response.Created(c, AttachmentResponse{ID: att.ID, DisplayName: att.DisplayName, OrderIndex: att.OrderIndex})
	case "save":
		var req SaveDraftRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		sub, err := h.assignments.SaveDraft(ctx, offeringID(c), targetID(c), middleware.GetUserID(c),
			classroom.SaveDraftInput{Content: req.Content, Files: fileRefs(req.Files)})
		if err != nil {
			h.respondError(c, err)
			return
		}
		response.OK(c, submissionResponse(sub))
	case "submit":
		sub, err := h.assignments.Submit(ctx, offeringID(c), targetID(c), middleware.GetUserID(c))
		if err != nil {
			h.respondError(c, err)
			return
		}
		response.OK(c, submissionResponse(sub))
	case "discard":
		if err := h.assignments.Discard(ctx, offeringID(c), targetID(c), middleware.GetUserID(c)); err != nil {
			h.respondError(c, err)
			return
		}
		response.NoContent(c)
	case "grade":
		var req GradeSubmissionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		sub, err := h.assignments.Grade(ctx, offeringID(c), targetID(c), req.StudentID,
			middleware.GetUserID(c), req.Score, req.Feedback)
		if err != nil {
			h.respondError(c, err)
			return
		}
		response.OK(c, submissionResponse(sub))
	default:
		response.NotFound(c, "unknown method")
	}
}

func (h *Handler) DownloadAssignmentAttachment(c *gin.Context) {
	attachmentID, err := uuid.Parse(c.Param("attachmentId"))
	if err != nil {
		response.NotFound(c, "attachment not found")
		return
	}
	url, err := h.assignments.PresignAttachment(c.Request.Context(), offeringID(c), targetID(c),
		attachmentID, studentView(c))
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.Redirect(http.StatusTemporaryRedirect, url)
}

func (h *Handler) DetachAssignmentFile(c *gin.Context) {
	attachmentID, err := uuid.Parse(c.Param("attachmentId"))
	if err != nil {
		response.NotFound(c, "attachment not found")
		return
	}
	if err := h.assignments.Detach(c.Request.Context(), offeringID(c), targetID(c), attachmentID); err != nil {
		h.respondError(c, err)
		return
	}
	response.NoContent(c)
}

// MySubmission is the caller's own draft or submitted work.
func (h *Handler) MySubmission(c *gin.Context) {
	sub, files, err := h.assignments.MySubmission(c.Request.Context(), offeringID(c), targetID(c), middleware.GetUserID(c))
	if err != nil {
		h.respondError(c, err)
		return
	}
	fileResponses := make([]AttachmentResponse, len(files))
	for i, f := range files {
		fileResponses[i] = AttachmentResponse{ID: f.ID, DisplayName: f.DisplayName, OrderIndex: f.OrderIndex}
	}
	response.OK(c, gin.H{"submission": submissionResponse(sub), "files": fileResponses})
}

// ListSubmissions is the teacher's view of everyone's work.
func (h *Handler) ListSubmissions(c *gin.Context) {
	if !requireTeaching(c) {
		return
	}
	subs, err := h.assignments.Submissions(c.Request.Context(), offeringID(c), targetID(c))
	if err != nil {
		h.respondError(c, err)
		return
	}
	result := make([]SubmissionWithStudentResponse, len(subs))
	for i, s := range subs {
		result[i] = SubmissionWithStudentResponse{
			SubmissionResponse: submissionResponse(&s.Submission),
			StudentName:        s.StudentName,
			StudentEmail:       s.StudentEmail,
			StudentAvatar:      s.StudentAvatar,
		}
	}
	response.OK(c, result)
}

// DownloadSubmissionFile serves a submission file to its owner or to
// teaching staff.
func (h *Handler) DownloadSubmissionFile(c *gin.Context) {
	studentID, err := uuid.Parse(c.Param("studentId"))
	if err != nil {
		response.NotFound(c, "submission not found")
		return
	}
	if !teaching(c) && studentID != middleware.GetUserID(c) {
		response.Forbidden(c, "permission denied")
		return
	}
	fileID, err := uuid.Parse(c.Param("fileId"))
	if err != nil {
		response.NotFound(c, "file not found")
		return
	}
	url, err := h.assignments.PresignSubmissionFile(c.Request.Context(), offeringID(c), targetID(c),
		studentID, fileID)
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.Redirect(http.StatusTemporaryRedirect, url)
}
