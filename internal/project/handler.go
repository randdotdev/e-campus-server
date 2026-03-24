package project

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ranjdotdev/e-campus-server/internal/middleware"
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

func (h *Handler) CreateProject(c *gin.Context) {
	offeringID, err := uuid.Parse(c.Param("offering_id"))
	if err != nil {
		response.BadRequest(c, "invalid offering id")
		return
	}

	var req CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	userID := middleware.GetUserID(c)

	isTeacher, err := h.service.IsTeacher(c.Request.Context(), offeringID, userID)
	if err != nil {
		h.log.Error("check teacher failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if !isTeacher {
		response.Forbidden(c, "teacher access required")
		return
	}

	p := &Project{
		OfferingID:           offeringID,
		Title:                req.Title,
		Body:                 req.Body,
		Deadline:             req.Deadline,
		MaxScore:             req.MaxScore,
		MinMembers:           req.MinMembers,
		MaxMembers:           req.MaxMembers,
		MergeTarget:          req.MergeTarget,
		RegistrationDeadline: req.RegistrationDeadline,
		Visibility:           req.Visibility,
		AllowLate:            req.AllowLate,
		PublishAt:            req.PublishAt,
		CreatedBy:            &userID,
	}

	if err := h.service.CreateProject(c.Request.Context(), p); err != nil {
		h.log.Error("create project failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.Created(c, ToProjectResponse(p, time.Now()))
}

func (h *Handler) GetProject(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid project id")
		return
	}

	p, err := h.service.GetProject(c.Request.Context(), id)
	if errors.Is(err, ErrProjectNotFound) {
		response.NotFound(c, "project not found")
		return
	}
	if err != nil {
		h.log.Error("get project failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	userID := middleware.GetUserID(c)
	now := time.Now()

	isTeacherOrAssistant, err := h.service.IsTeacherOrAssistant(c.Request.Context(), p.OfferingID, userID)
	if err != nil {
		h.log.Error("check teacher failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	if !isTeacherOrAssistant {
		if !IsPublished(p.PublishAt, now) {
			response.NotFound(c, "project not found")
			return
		}

		enrolled, err := h.service.IsEnrolled(c.Request.Context(), p.OfferingID, userID)
		if err != nil {
			h.log.Error("check enrollment failed", zap.Error(err))
			response.InternalError(c)
			return
		}
		if !enrolled {
			response.Forbidden(c, "not enrolled")
			return
		}
	}

	attachments, err := h.service.GetAttachments(c.Request.Context(), id)
	if err != nil {
		h.log.Error("get attachments failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	resp := struct {
		ProjectResponse
		Attachments []AttachmentResponse `json:"attachments"`
	}{
		ProjectResponse: ToProjectResponse(p, now),
		Attachments:     ToAttachmentsResponse(attachments),
	}

	response.OK(c, resp)
}

func (h *Handler) ListProjects(c *gin.Context) {
	offeringID, err := uuid.Parse(c.Param("offering_id"))
	if err != nil {
		response.BadRequest(c, "invalid offering id")
		return
	}

	userID := middleware.GetUserID(c)
	now := time.Now()

	isTeacherOrAssistant, err := h.service.IsTeacherOrAssistant(c.Request.Context(), offeringID, userID)
	if err != nil {
		h.log.Error("check teacher failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	var projects []Project
	if isTeacherOrAssistant {
		projects, err = h.service.ListProjects(c.Request.Context(), offeringID)
	} else {
		enrolled, enrollErr := h.service.IsEnrolled(c.Request.Context(), offeringID, userID)
		if enrollErr != nil {
			h.log.Error("check enrollment failed", zap.Error(enrollErr))
			response.InternalError(c)
			return
		}
		if !enrolled {
			response.Forbidden(c, "not enrolled")
			return
		}
		projects, err = h.service.ListPublishedProjects(c.Request.Context(), offeringID)
	}

	if err != nil {
		h.log.Error("list projects failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, ToProjectsResponse(projects, now))
}

func (h *Handler) UpdateProject(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid project id")
		return
	}

	p, err := h.service.GetProject(c.Request.Context(), id)
	if errors.Is(err, ErrProjectNotFound) {
		response.NotFound(c, "project not found")
		return
	}
	if err != nil {
		h.log.Error("get project failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	userID := middleware.GetUserID(c)

	isTeacher, err := h.service.IsTeacher(c.Request.Context(), p.OfferingID, userID)
	if err != nil {
		h.log.Error("check teacher failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if !isTeacher {
		response.Forbidden(c, "teacher access required")
		return
	}

	var req UpdateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	updated, err := h.service.UpdateProject(c.Request.Context(), id, ToProjectUpdates(req))
	if err != nil {
		h.log.Error("update project failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, ToProjectResponse(updated, time.Now()))
}

func (h *Handler) DeleteProject(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid project id")
		return
	}

	p, err := h.service.GetProject(c.Request.Context(), id)
	if errors.Is(err, ErrProjectNotFound) {
		response.NotFound(c, "project not found")
		return
	}
	if err != nil {
		h.log.Error("get project failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	userID := middleware.GetUserID(c)

	isTeacher, err := h.service.IsTeacher(c.Request.Context(), p.OfferingID, userID)
	if err != nil {
		h.log.Error("check teacher failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if !isTeacher {
		response.Forbidden(c, "teacher access required")
		return
	}

	if err := h.service.DeleteProject(c.Request.Context(), id); err != nil {
		h.log.Error("delete project failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.NoContent(c)
}

func (h *Handler) Register(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid project id")
		return
	}

	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	err = h.service.Register(c.Request.Context(), id, req.TeamID, req.ProjectTitle)
	if errors.Is(err, ErrProjectNotFound) {
		response.NotFound(c, "project not found")
	} else if errors.Is(err, ErrNotPublished) {
		response.BadRequest(c, "project not published")
	} else if errors.Is(err, ErrRegistrationClosed) {
		response.BadRequest(c, "registration deadline passed")
	} else if errors.Is(err, ErrAlreadyRegistered) {
		response.Conflict(c, "team already registered")
	} else if errors.Is(err, ErrTeamTooSmall) {
		response.BadRequest(c, "team has fewer than minimum members")
	} else if errors.Is(err, ErrTeamTooLarge) {
		response.BadRequest(c, "team has more than maximum members")
	} else if errors.Is(err, ErrMembersNotEnrolled) {
		response.BadRequest(c, "some team members not enrolled in course")
	} else if err != nil {
		h.log.Error("register failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.Created(c, gin.H{"registered": true})
	}
}

func (h *Handler) Unregister(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid project id")
		return
	}

	teamID, err := uuid.Parse(c.Param("team_id"))
	if err != nil {
		response.BadRequest(c, "invalid team id")
		return
	}

	err = h.service.Unregister(c.Request.Context(), id, teamID)
	if errors.Is(err, ErrProjectNotFound) {
		response.NotFound(c, "project not found")
	} else if errors.Is(err, ErrNotRegistered) {
		response.NotFound(c, "team not registered")
	} else if err != nil {
		h.log.Error("unregister failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.NoContent(c)
	}
}

func (h *Handler) GetRegistrations(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid project id")
		return
	}

	registrations, err := h.service.GetRegistrations(c.Request.Context(), id)
	if err != nil {
		h.log.Error("get registrations failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, ToRegistrationsResponse(registrations))
}

func (h *Handler) GetProjectGroups(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid project id")
		return
	}

	groups, err := h.service.GetProjectGroups(c.Request.Context(), id)
	if err != nil {
		h.log.Error("get groups failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, ToProjectGroupsResponse(groups))
}

func (h *Handler) GetMyProjectGroup(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid project id")
		return
	}

	userID := middleware.GetUserID(c)

	group, err := h.service.GetMyProjectGroup(c.Request.Context(), id, userID)
	if err != nil {
		h.log.Error("get my group failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if group == nil {
		response.NotFound(c, "not in any group for this project")
		return
	}

	response.OK(c, ToProjectGroupResponse(group))
}

func (h *Handler) CreateProjectGroups(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid project id")
		return
	}

	p, err := h.service.GetProject(c.Request.Context(), id)
	if errors.Is(err, ErrProjectNotFound) {
		response.NotFound(c, "project not found")
		return
	}
	if err != nil {
		h.log.Error("get project failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	userID := middleware.GetUserID(c)

	isTeacher, err := h.service.IsTeacher(c.Request.Context(), p.OfferingID, userID)
	if err != nil {
		h.log.Error("check teacher failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if !isTeacher {
		response.Forbidden(c, "teacher access required")
		return
	}

	if err := h.service.CreateProjectGroups(c.Request.Context(), id); err != nil {
		h.log.Error("create groups failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	groups, _ := h.service.GetProjectGroups(c.Request.Context(), id)
	response.Created(c, ToProjectGroupsResponse(groups))
}

func (h *Handler) CreateSubmission(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid project id")
		return
	}

	groupID, err := uuid.Parse(c.Param("group_id"))
	if err != nil {
		response.BadRequest(c, "invalid group id")
		return
	}

	var req CreateSubmissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	userID := middleware.GetUserID(c)

	sub, err := h.service.CreateSubmission(c.Request.Context(), projectID, groupID, userID, req.Content, ToFileInputs(req.Files))
	if errors.Is(err, ErrProjectNotFound) {
		response.NotFound(c, "project not found")
	} else if errors.Is(err, ErrGroupNotFound) {
		response.NotFound(c, "group not found")
	} else if errors.Is(err, ErrNotGroupLeader) {
		response.Forbidden(c, "only group leader can submit")
	} else if errors.Is(err, ErrAlreadySubmitted) {
		response.Conflict(c, "submission already exists")
	} else if errors.Is(err, ErrFileNotOwned) {
		response.BadRequest(c, "file not owned by user")
	} else if err != nil {
		h.log.Error("create submission failed", zap.Error(err))
		response.InternalError(c)
	} else {
		p, _ := h.service.GetProject(c.Request.Context(), projectID)
		files, _ := h.service.GetSubmissionFiles(c.Request.Context(), sub.ID)
		response.Created(c, ToSubmissionResponse(sub, files, p.Deadline))
	}
}

func (h *Handler) UpdateSubmission(c *gin.Context) {
	id, err := uuid.Parse(c.Param("submission_id"))
	if err != nil {
		response.BadRequest(c, "invalid submission id")
		return
	}

	var req UpdateSubmissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	userID := middleware.GetUserID(c)

	sub, err := h.service.UpdateSubmission(c.Request.Context(), id, userID, req.Content, ToFileInputs(req.Files))
	if errors.Is(err, ErrSubmissionNotFound) {
		response.NotFound(c, "submission not found")
	} else if errors.Is(err, ErrAlreadySubmitted) {
		response.Conflict(c, "cannot update submitted submission")
	} else if errors.Is(err, ErrNotGroupLeader) {
		response.Forbidden(c, "only group leader can update")
	} else if errors.Is(err, ErrFileNotOwned) {
		response.BadRequest(c, "file not owned by user")
	} else if err != nil {
		h.log.Error("update submission failed", zap.Error(err))
		response.InternalError(c)
	} else {
		p, _ := h.service.GetProject(c.Request.Context(), sub.ProjectID)
		files, _ := h.service.GetSubmissionFiles(c.Request.Context(), sub.ID)
		response.OK(c, ToSubmissionResponse(sub, files, p.Deadline))
	}
}

func (h *Handler) SubmitSubmission(c *gin.Context) {
	id, err := uuid.Parse(c.Param("submission_id"))
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
	} else if errors.Is(err, ErrNotGroupLeader) {
		response.Forbidden(c, "only group leader can submit")
	} else if errors.Is(err, ErrSubmissionsClosed) {
		response.BadRequest(c, "submission deadline passed")
	} else if errors.Is(err, ErrNoContent) {
		response.BadRequest(c, "submission must have content or files")
	} else if err != nil {
		h.log.Error("submit submission failed", zap.Error(err))
		response.InternalError(c)
	} else {
		p, _ := h.service.GetProject(c.Request.Context(), sub.ProjectID)
		files, _ := h.service.GetSubmissionFiles(c.Request.Context(), sub.ID)
		response.OK(c, ToSubmissionResponse(sub, files, p.Deadline))
	}
}

func (h *Handler) GetMySubmission(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid project id")
		return
	}

	userID := middleware.GetUserID(c)

	sub, err := h.service.GetMySubmission(c.Request.Context(), projectID, userID)
	if errors.Is(err, ErrNotGroupMember) {
		response.NotFound(c, "not in any group for this project")
		return
	}
	if errors.Is(err, ErrSubmissionNotFound) {
		response.NotFound(c, "no submission yet")
		return
	}
	if err != nil {
		h.log.Error("get my submission failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	p, _ := h.service.GetProject(c.Request.Context(), projectID)
	files, _ := h.service.GetSubmissionFiles(c.Request.Context(), sub.ID)

	myGrade, _ := h.service.GetMyGrade(c.Request.Context(), sub.ID, userID)

	resp := struct {
		SubmissionResponse
		MyGrade MyGradeResponse `json:"my_grade"`
	}{
		SubmissionResponse: ToSubmissionResponse(sub, files, p.Deadline),
		MyGrade:            ToMyGradeResponse(myGrade, p.ScoresPublic),
	}

	response.OK(c, resp)
}

func (h *Handler) ListSubmissions(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid project id")
		return
	}

	p, err := h.service.GetProject(c.Request.Context(), projectID)
	if errors.Is(err, ErrProjectNotFound) {
		response.NotFound(c, "project not found")
		return
	}
	if err != nil {
		h.log.Error("get project failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	userID := middleware.GetUserID(c)

	isTeacherOrAssistant, err := h.service.IsTeacherOrAssistant(c.Request.Context(), p.OfferingID, userID)
	if err != nil {
		h.log.Error("check teacher failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if !isTeacherOrAssistant {
		response.Forbidden(c, "teacher or assistant access required")
		return
	}

	submissions, err := h.service.ListSubmissions(c.Request.Context(), projectID)
	if err != nil {
		h.log.Error("list submissions failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	result := make([]SubmissionTeacherResponse, len(submissions))
	for i := range submissions {
		files, _ := h.service.GetSubmissionFiles(c.Request.Context(), submissions[i].ID)
		grades, _ := h.service.GetGrades(c.Request.Context(), submissions[i].ID)
		result[i] = ToSubmissionTeacherResponse(&submissions[i], files, grades, p.Deadline)
	}

	response.OK(c, result)
}

func (h *Handler) GradeSubmission(c *gin.Context) {
	submissionID, err := uuid.Parse(c.Param("submission_id"))
	if err != nil {
		response.BadRequest(c, "invalid submission id")
		return
	}

	studentID, err := uuid.Parse(c.Param("student_id"))
	if err != nil {
		response.BadRequest(c, "invalid student id")
		return
	}

	sub, err := h.service.GetSubmission(c.Request.Context(), submissionID)
	if err != nil {
		h.log.Error("get submission failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if sub == nil {
		response.NotFound(c, "submission not found")
		return
	}

	p, err := h.service.GetProject(c.Request.Context(), sub.ProjectID)
	if err != nil {
		h.log.Error("get project failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	userID := middleware.GetUserID(c)

	isTeacherOrAssistant, err := h.service.IsTeacherOrAssistant(c.Request.Context(), p.OfferingID, userID)
	if err != nil {
		h.log.Error("check teacher failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if !isTeacherOrAssistant {
		response.Forbidden(c, "teacher or assistant access required")
		return
	}

	var req GradeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	err = h.service.GradeSubmission(c.Request.Context(), submissionID, studentID, userID, req.Score, req.Feedback)
	if errors.Is(err, ErrInvalidScore) {
		response.BadRequest(c, "score out of range")
	} else if err != nil {
		h.log.Error("grade submission failed", zap.Error(err))
		response.InternalError(c)
	} else {
		grade, _ := h.service.GetMyGrade(c.Request.Context(), submissionID, studentID)
		response.OK(c, GradeResponse{
			StudentID: studentID,
			Score:     grade.Score,
			Feedback:  grade.Feedback,
			GradedAt:  grade.GradedAt,
		})
	}
}

func (h *Handler) PublishScores(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid project id")
		return
	}

	p, err := h.service.GetProject(c.Request.Context(), id)
	if errors.Is(err, ErrProjectNotFound) {
		response.NotFound(c, "project not found")
		return
	}
	if err != nil {
		h.log.Error("get project failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	userID := middleware.GetUserID(c)

	isTeacher, err := h.service.IsTeacher(c.Request.Context(), p.OfferingID, userID)
	if err != nil {
		h.log.Error("check teacher failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if !isTeacher {
		response.Forbidden(c, "teacher access required")
		return
	}

	if err := h.service.PublishScores(c.Request.Context(), id); err != nil {
		h.log.Error("publish scores failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, gin.H{"scores_public": true})
}

func (h *Handler) AddAttachment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid project id")
		return
	}

	p, err := h.service.GetProject(c.Request.Context(), id)
	if errors.Is(err, ErrProjectNotFound) {
		response.NotFound(c, "project not found")
		return
	}
	if err != nil {
		h.log.Error("get project failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	userID := middleware.GetUserID(c)

	isTeacher, err := h.service.IsTeacher(c.Request.Context(), p.OfferingID, userID)
	if err != nil {
		h.log.Error("check teacher failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if !isTeacher {
		response.Forbidden(c, "teacher access required")
		return
	}

	var req AddAttachmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	att := &ProjectAttachment{
		ProjectID:    id,
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
	projectID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid project id")
		return
	}

	attachmentID, err := uuid.Parse(c.Param("attachment_id"))
	if err != nil {
		response.BadRequest(c, "invalid attachment id")
		return
	}

	p, err := h.service.GetProject(c.Request.Context(), projectID)
	if errors.Is(err, ErrProjectNotFound) {
		response.NotFound(c, "project not found")
		return
	}
	if err != nil {
		h.log.Error("get project failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	userID := middleware.GetUserID(c)

	isTeacher, err := h.service.IsTeacher(c.Request.Context(), p.OfferingID, userID)
	if err != nil {
		h.log.Error("check teacher failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if !isTeacher {
		response.Forbidden(c, "teacher access required")
		return
	}

	if err := h.service.DeleteAttachment(c.Request.Context(), attachmentID); err != nil {
		h.log.Error("delete attachment failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.NoContent(c)
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	projects := r.Group("/projects")
	projects.Use(authMiddleware)
	{
		projects.GET("/:id", h.GetProject)
		projects.PUT("/:id", h.UpdateProject)
		projects.DELETE("/:id", h.DeleteProject)
		projects.POST("/:id/attachments", h.AddAttachment)
		projects.DELETE("/:id/attachments/:attachment_id", h.DeleteAttachment)
		projects.POST("/:id/register", h.Register)
		projects.DELETE("/:id/register/:team_id", h.Unregister)
		projects.GET("/:id/registrations", h.GetRegistrations)
		projects.GET("/:id/groups", h.GetProjectGroups)
		projects.GET("/:id/groups/me", h.GetMyProjectGroup)
		projects.POST("/:id/groups", h.CreateProjectGroups)
		projects.POST("/:id/groups/:group_id/submissions", h.CreateSubmission)
		projects.GET("/:id/submissions", h.ListSubmissions)
		projects.GET("/:id/submissions/me", h.GetMySubmission)
		projects.PUT("/:id/publish-scores", h.PublishScores)
	}

	submissions := r.Group("/project-submissions")
	submissions.Use(authMiddleware)
	{
		submissions.PUT("/:submission_id", h.UpdateSubmission)
		submissions.POST("/:submission_id/submit", h.SubmitSubmission)
		submissions.PUT("/:submission_id/grades/:student_id", h.GradeSubmission)
	}

	offerings := r.Group("/offerings")
	offerings.Use(authMiddleware)
	{
		offerings.GET("/:offering_id/projects", h.ListProjects)
		offerings.POST("/:offering_id/projects", h.CreateProject)
	}
}
