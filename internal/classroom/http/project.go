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

type ProjectResponse struct {
	ID                   uuid.UUID  `json:"id"`
	OfferingID           uuid.UUID  `json:"offering_id"`
	Title                string     `json:"title"`
	Body                 *string    `json:"body"`
	Deadline             time.Time  `json:"deadline"`
	MaxScore             float64    `json:"max_score"`
	MinMembers           int        `json:"min_members"`
	MaxMembers           int        `json:"max_members"`
	MergeTarget          *int       `json:"merge_target"`
	RegistrationDeadline *time.Time `json:"registration_deadline"`
	Visibility           string     `json:"visibility"`
	AllowLate            bool       `json:"allow_late"`
	PublishAt            *time.Time `json:"publish_at"`
	ScoresPublic         bool       `json:"scores_public"`
	Version              int64      `json:"version"`
	CreatedAt            time.Time  `json:"created_at"`
}

func projectResponse(p *classroom.Project) ProjectResponse {
	return ProjectResponse{
		ID: p.ID, OfferingID: p.OfferingID, Title: p.Title, Body: p.Body,
		Deadline: p.Deadline, MaxScore: p.MaxScore,
		MinMembers: p.MinMembers, MaxMembers: p.MaxMembers, MergeTarget: p.MergeTarget,
		RegistrationDeadline: p.RegistrationDeadline, Visibility: string(p.Visibility),
		AllowLate: p.AllowLate, PublishAt: p.PublishAt, ScoresPublic: p.ScoresPublic,
		Version: p.Version, CreatedAt: p.CreatedAt,
	}
}

type ProjectSubmissionResponse struct {
	ID             uuid.UUID  `json:"id"`
	ProjectID      uuid.UUID  `json:"project_id"`
	ProjectGroupID uuid.UUID  `json:"project_group_id"`
	Content        *string    `json:"content"`
	SubmittedAt    *time.Time `json:"submitted_at"`
	SubmittedBy    *uuid.UUID `json:"submitted_by"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      *time.Time `json:"updated_at"`
}

func projectSubmissionResponse(s *classroom.ProjectSubmission) ProjectSubmissionResponse {
	return ProjectSubmissionResponse{
		ID: s.ID, ProjectID: s.ProjectID, ProjectGroupID: s.ProjectGroupID,
		Content: s.Content, SubmittedAt: s.SubmittedAt, SubmittedBy: s.SubmittedBy,
		CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}
}

type ProjectGroupResponse struct {
	ID           uuid.UUID `json:"id"`
	Name         *string   `json:"name"`
	ProjectTitle *string   `json:"project_title"`
	LeaderID     uuid.UUID `json:"leader_id"`
	Finalized    bool      `json:"finalized"`
	Members      []gin.H   `json:"members"`
}

func projectGroupResponse(g *classroom.ProjectGroupWithMembers) ProjectGroupResponse {
	members := make([]gin.H, len(g.Members))
	for i, m := range g.Members {
		members[i] = gin.H{"user_id": m.UserID, "name": m.Name, "username": m.Username}
	}
	return ProjectGroupResponse{
		ID: g.ID, Name: g.Name, ProjectTitle: g.ProjectTitle,
		LeaderID: g.LeaderID, Finalized: g.Finalized, Members: members,
	}
}

type CreateProjectRequest struct {
	Title                string     `json:"title" binding:"required,max=255"`
	Body                 *string    `json:"body"`
	Deadline             time.Time  `json:"deadline" binding:"required"`
	MaxScore             float64    `json:"max_score" binding:"required,gt=0"`
	MinMembers           int        `json:"min_members" binding:"required,gte=1"`
	MaxMembers           int        `json:"max_members" binding:"required,gte=1"`
	MergeTarget          *int       `json:"merge_target" binding:"omitempty,gte=1"`
	RegistrationDeadline *time.Time `json:"registration_deadline"`
	Visibility           string     `json:"visibility" binding:"omitempty,oneof=hidden registered all"`
	AllowLate            bool       `json:"allow_late"`
	PublishAt            *time.Time `json:"publish_at"`
}

type UpdateProjectRequest struct {
	Title                *string    `json:"title" binding:"omitempty,max=255"`
	Body                 *string    `json:"body"`
	Deadline             *time.Time `json:"deadline"`
	MaxScore             *float64   `json:"max_score" binding:"omitempty,gt=0"`
	MinMembers           *int       `json:"min_members" binding:"omitempty,gte=1"`
	MaxMembers           *int       `json:"max_members" binding:"omitempty,gte=1"`
	MergeTarget          *int       `json:"merge_target" binding:"omitempty,gte=1"`
	RegistrationDeadline *time.Time `json:"registration_deadline"`
	Visibility           *string    `json:"visibility" binding:"omitempty,oneof=hidden registered all"`
	AllowLate            *bool      `json:"allow_late"`
	PublishAt            *time.Time `json:"publish_at"`
	ScoresPublic         *bool      `json:"scores_public"`
}

func (h *Handler) CreateProject(c *gin.Context) {
	var req CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	p, err := h.projects.Create(c.Request.Context(), classroom.CreateProjectInput{
		OfferingID: offeringID(c), CreatedBy: middleware.GetUserID(c),
		Title: req.Title, Body: req.Body, Deadline: req.Deadline, MaxScore: req.MaxScore,
		MinMembers: req.MinMembers, MaxMembers: req.MaxMembers, MergeTarget: req.MergeTarget,
		RegistrationDeadline: req.RegistrationDeadline,
		Visibility:           classroom.ProjectVisibility(req.Visibility),
		AllowLate:            req.AllowLate, PublishAt: req.PublishAt,
	})
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Created(c, projectResponse(p))
}

func (h *Handler) GetProject(c *gin.Context) {
	p, attachments, err := h.projects.Get(c.Request.Context(), offeringID(c), targetID(c), studentView(c))
	if err != nil {
		h.respondError(c, err)
		return
	}
	atts := make([]AttachmentResponse, len(attachments))
	for i, a := range attachments {
		atts[i] = AttachmentResponse{ID: a.ID, DisplayName: a.DisplayName, OrderIndex: a.OrderIndex}
	}
	response.OK(c, gin.H{"project": projectResponse(p), "attachments": atts})
}

func (h *Handler) ListProjects(c *gin.Context) {
	projects, err := h.projects.List(c.Request.Context(), offeringID(c), studentView(c))
	if err != nil {
		h.respondError(c, err)
		return
	}
	result := make([]ProjectResponse, len(projects))
	for i := range projects {
		result[i] = projectResponse(&projects[i])
	}
	response.OK(c, result)
}

func (h *Handler) UpdateProject(c *gin.Context) {
	var req UpdateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	in := classroom.UpdateProjectInput{
		Title: req.Title, Body: req.Body, Deadline: req.Deadline, MaxScore: req.MaxScore,
		MinMembers: req.MinMembers, MaxMembers: req.MaxMembers, MergeTarget: req.MergeTarget,
		RegistrationDeadline: req.RegistrationDeadline,
		AllowLate:            req.AllowLate, PublishAt: req.PublishAt, ScoresPublic: req.ScoresPublic,
	}
	if req.Visibility != nil {
		v := classroom.ProjectVisibility(*req.Visibility)
		in.Visibility = &v
	}
	p, err := h.projects.Update(c.Request.Context(), offeringID(c), targetID(c), in)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, projectResponse(p))
}

func (h *Handler) DeleteProject(c *gin.Context) {
	if err := h.projects.Delete(c.Request.Context(), offeringID(c), targetID(c)); err != nil {
		h.respondError(c, err)
		return
	}
	response.NoContent(c)
}

// ProjectCustom dispatches the project's colon methods.
func (h *Handler) ProjectCustom(c *gin.Context) {
	ctx := c.Request.Context()
	actor := middleware.GetUserID(c)
	switch customAction(c) {
	case "attach":
		var req AttachRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		att, err := h.projects.Attach(ctx, offeringID(c), targetID(c), actor,
			classroom.FileRef{UploadID: req.UploadID, DisplayName: req.DisplayName})
		if err != nil {
			h.respondError(c, err)
			return
		}
		response.Created(c, AttachmentResponse{ID: att.ID, DisplayName: att.DisplayName, OrderIndex: att.OrderIndex})
	case "register":
		var req struct {
			TeamID       uuid.UUID `json:"team_id" binding:"required"`
			ProjectTitle string    `json:"project_title" binding:"required,max=255"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		if err := h.projects.Register(ctx, offeringID(c), targetID(c), req.TeamID, actor, req.ProjectTitle); err != nil {
			h.respondError(c, err)
			return
		}
		response.OK(c, gin.H{"registered": true})
	case "unregister":
		var req struct {
			TeamID uuid.UUID `json:"team_id" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		if err := h.projects.Unregister(ctx, offeringID(c), targetID(c), req.TeamID, actor); err != nil {
			h.respondError(c, err)
			return
		}
		response.NoContent(c)
	case "formGroups":
		formed, unmerged, err := h.projects.FormGroups(ctx, offeringID(c), targetID(c))
		if err != nil {
			h.respondError(c, err)
			return
		}
		response.OK(c, gin.H{"formed": formed, "unmerged_teams": unmerged})
	case "save":
		var req SaveDraftRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		sub, err := h.projects.SaveSubmission(ctx, offeringID(c), targetID(c), actor, req.Content, fileRefs(req.Files))
		if err != nil {
			h.respondError(c, err)
			return
		}
		response.OK(c, projectSubmissionResponse(sub))
	case "submit":
		sub, err := h.projects.Submit(ctx, offeringID(c), targetID(c), actor)
		if err != nil {
			h.respondError(c, err)
			return
		}
		response.OK(c, projectSubmissionResponse(sub))
	case "grade":
		var req struct {
			SubmissionID uuid.UUID `json:"submission_id" binding:"required"`
			StudentID    uuid.UUID `json:"student_id" binding:"required"`
			Score        float64   `json:"score"`
			Feedback     *string   `json:"feedback"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		if err := h.projects.Grade(ctx, offeringID(c), targetID(c), req.SubmissionID,
			req.StudentID, actor, req.Score, req.Feedback); err != nil {
			h.respondError(c, err)
			return
		}
		response.OK(c, gin.H{"graded": req.StudentID})
	default:
		response.NotFound(c, "unknown method")
	}
}

func (h *Handler) DownloadProjectAttachment(c *gin.Context) {
	attachmentID, err := uuid.Parse(c.Param("attachmentId"))
	if err != nil {
		response.NotFound(c, "attachment not found")
		return
	}
	url, err := h.projects.PresignAttachment(c.Request.Context(), offeringID(c), targetID(c),
		attachmentID, studentView(c))
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.Redirect(http.StatusTemporaryRedirect, url)
}

func (h *Handler) DetachProjectFile(c *gin.Context) {
	attachmentID, err := uuid.Parse(c.Param("attachmentId"))
	if err != nil {
		response.NotFound(c, "attachment not found")
		return
	}
	if err := h.projects.Detach(c.Request.Context(), offeringID(c), targetID(c), attachmentID); err != nil {
		h.respondError(c, err)
		return
	}
	response.NoContent(c)
}

func (h *Handler) ListRegistrations(c *gin.Context) {
	regs, err := h.projects.Registrations(c.Request.Context(), offeringID(c), targetID(c),
		middleware.GetUserID(c), teaching(c))
	if err != nil {
		h.respondError(c, err)
		return
	}
	type row struct {
		ID           uuid.UUID `json:"id"`
		TeamID       uuid.UUID `json:"team_id"`
		TeamName     *string   `json:"team_name"`
		LeaderID     uuid.UUID `json:"leader_id"`
		MemberCount  int       `json:"member_count"`
		ProjectTitle string    `json:"project_title"`
		RegisteredAt time.Time `json:"registered_at"`
	}
	result := make([]row, len(regs))
	for i, r := range regs {
		result[i] = row{r.ID, r.TeamID, r.TeamName, r.LeaderID, r.MemberCount, r.ProjectTitle, r.RegisteredAt}
	}
	response.OK(c, result)
}

func (h *Handler) ListGroups(c *gin.Context) {
	if !requireTeaching(c) {
		return
	}
	groups, err := h.projects.Groups(c.Request.Context(), offeringID(c), targetID(c))
	if err != nil {
		h.respondError(c, err)
		return
	}
	result := make([]ProjectGroupResponse, len(groups))
	for i := range groups {
		result[i] = projectGroupResponse(&groups[i])
	}
	response.OK(c, result)
}

func (h *Handler) MyGroup(c *gin.Context) {
	group, err := h.projects.MyGroup(c.Request.Context(), offeringID(c), targetID(c), middleware.GetUserID(c))
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, projectGroupResponse(group))
}

func (h *Handler) MyProjectSubmission(c *gin.Context) {
	sub, files, err := h.projects.MySubmission(c.Request.Context(), offeringID(c), targetID(c), middleware.GetUserID(c))
	if err != nil {
		h.respondError(c, err)
		return
	}
	fileResponses := make([]AttachmentResponse, len(files))
	for i, f := range files {
		fileResponses[i] = AttachmentResponse{ID: f.ID, DisplayName: f.DisplayName, OrderIndex: f.OrderIndex}
	}
	response.OK(c, gin.H{"submission": projectSubmissionResponse(sub), "files": fileResponses})
}

func (h *Handler) ListProjectSubmissions(c *gin.Context) {
	if !requireTeaching(c) {
		return
	}
	subs, err := h.projects.Submissions(c.Request.Context(), offeringID(c), targetID(c))
	if err != nil {
		h.respondError(c, err)
		return
	}
	result := make([]ProjectSubmissionResponse, len(subs))
	for i := range subs {
		result[i] = projectSubmissionResponse(&subs[i])
	}
	response.OK(c, result)
}

// DownloadProjectSubmissionFile serves a submission file to the group's
// members or teaching staff.
func (h *Handler) DownloadProjectSubmissionFile(c *gin.Context) {
	submissionID, err := uuid.Parse(c.Param("submissionId"))
	if err != nil {
		response.NotFound(c, "submission not found")
		return
	}
	if !teaching(c) {
		sub, _, err := h.projects.MySubmission(c.Request.Context(), offeringID(c), targetID(c), middleware.GetUserID(c))
		if err != nil || sub.ID != submissionID {
			response.Forbidden(c, "permission denied")
			return
		}
	}
	fileID, err := uuid.Parse(c.Param("fileId"))
	if err != nil {
		response.NotFound(c, "file not found")
		return
	}
	url, err := h.projects.PresignSubmissionFile(c.Request.Context(), offeringID(c), targetID(c),
		submissionID, fileID)
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.Redirect(http.StatusTemporaryRedirect, url)
}

func (h *Handler) ListProjectGrades(c *gin.Context) {
	if !requireTeaching(c) {
		return
	}
	submissionID, err := uuid.Parse(c.Param("submissionId"))
	if err != nil {
		response.NotFound(c, "submission not found")
		return
	}
	grades, err := h.projects.Grades(c.Request.Context(), offeringID(c), targetID(c), submissionID)
	if err != nil {
		h.respondError(c, err)
		return
	}
	type row struct {
		StudentID   uuid.UUID  `json:"student_id"`
		StudentName string     `json:"student_name"`
		Score       *float64   `json:"score"`
		Feedback    *string    `json:"feedback"`
		GradedAt    *time.Time `json:"graded_at"`
	}
	result := make([]row, len(grades))
	for i, g := range grades {
		result[i] = row{g.StudentID, g.StudentName, g.Score, g.Feedback, g.GradedAt}
	}
	response.OK(c, result)
}

func (h *Handler) MyProjectGrade(c *gin.Context) {
	grade, err := h.projects.MyGrade(c.Request.Context(), offeringID(c), targetID(c), middleware.GetUserID(c))
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, gin.H{"score": grade.Score, "feedback": grade.Feedback, "graded_at": grade.GradedAt})
}
