package project

import (
	"time"

	"github.com/google/uuid"
)

type CreateProjectRequest struct {
	Title                string     `json:"title" validate:"required,max=255"`
	Body                 *string    `json:"body"`
	Deadline             time.Time  `json:"deadline" validate:"required"`
	MaxScore             float64    `json:"max_score" validate:"required,gt=0"`
	MinMembers           int        `json:"min_members" validate:"required,gte=1"`
	MaxMembers           int        `json:"max_members" validate:"required,gte=1"`
	MergeTarget          *int       `json:"merge_target"`
	RegistrationDeadline *time.Time `json:"registration_deadline"`
	Visibility           string     `json:"visibility" validate:"required,oneof=hidden registered all"`
	AllowLate            bool       `json:"allow_late"`
	PublishAt            *time.Time `json:"publish_at"`
}

type UpdateProjectRequest struct {
	Title                *string    `json:"title" validate:"omitempty,max=255"`
	Body                 *string    `json:"body"`
	Deadline             *time.Time `json:"deadline"`
	MaxScore             *float64   `json:"max_score" validate:"omitempty,gt=0"`
	MinMembers           *int       `json:"min_members" validate:"omitempty,gte=1"`
	MaxMembers           *int       `json:"max_members" validate:"omitempty,gte=1"`
	MergeTarget          *int       `json:"merge_target"`
	RegistrationDeadline *time.Time `json:"registration_deadline"`
	Visibility           *string    `json:"visibility" validate:"omitempty,oneof=hidden registered all"`
	AllowLate            *bool      `json:"allow_late"`
	PublishAt            *time.Time `json:"publish_at"`
}

type RegisterRequest struct {
	TeamID       uuid.UUID `json:"team_id" validate:"required"`
	ProjectTitle string    `json:"project_title" validate:"required,max=255"`
}

type AddAttachmentRequest struct {
	StoredFileID uuid.UUID `json:"stored_file_id" validate:"required"`
	DisplayName  string    `json:"display_name" validate:"required,max=255"`
	OrderIndex   int       `json:"order_index"`
}

type CreateSubmissionRequest struct {
	Content *string            `json:"content"`
	Files   []FileInputRequest `json:"files"`
}

type UpdateSubmissionRequest struct {
	Content *string            `json:"content"`
	Files   []FileInputRequest `json:"files"`
}

type FileInputRequest struct {
	StoredFileID uuid.UUID `json:"stored_file_id" validate:"required"`
	DisplayName  string    `json:"display_name" validate:"required,max=255"`
}

type GradeRequest struct {
	Score    float64 `json:"score" validate:"gte=0"`
	Feedback *string `json:"feedback"`
}

type ProjectResponse struct {
	ID                   uuid.UUID  `json:"id"`
	OfferingID           uuid.UUID  `json:"offering_id"`
	Title                string     `json:"title"`
	Body                 *string    `json:"body,omitempty"`
	Deadline             time.Time  `json:"deadline"`
	MaxScore             float64    `json:"max_score"`
	MinMembers           int        `json:"min_members"`
	MaxMembers           int        `json:"max_members"`
	MergeTarget          *int       `json:"merge_target,omitempty"`
	RegistrationDeadline *time.Time `json:"registration_deadline,omitempty"`
	Visibility           string     `json:"visibility"`
	AllowLate            bool       `json:"allow_late"`
	PublishAt            *time.Time `json:"publish_at,omitempty"`
	ScoresPublic         bool       `json:"scores_public"`
	IsPublished          bool       `json:"is_published"`
	IsRegistrationOpen   bool       `json:"is_registration_open"`
	IsDeadlinePassed     bool       `json:"is_deadline_passed"`
	CreatedAt            time.Time  `json:"created_at"`
}

type AttachmentResponse struct {
	ID           uuid.UUID `json:"id"`
	StoredFileID uuid.UUID `json:"stored_file_id"`
	DisplayName  string    `json:"display_name"`
	OrderIndex   int       `json:"order_index"`
	CreatedAt    time.Time `json:"created_at"`
}

type RegistrationResponse struct {
	ID           uuid.UUID `json:"id"`
	ProjectID    uuid.UUID `json:"project_id"`
	TeamID       uuid.UUID `json:"team_id"`
	TeamName     *string   `json:"team_name,omitempty"`
	LeaderID     uuid.UUID `json:"leader_id"`
	LeaderName   string    `json:"leader_name"`
	MemberCount  int       `json:"member_count"`
	ProjectTitle string    `json:"project_title"`
	RegisteredAt time.Time `json:"registered_at"`
}

type ProjectGroupResponse struct {
	ID           uuid.UUID             `json:"id"`
	ProjectID    uuid.UUID             `json:"project_id"`
	Name         *string               `json:"name,omitempty"`
	ProjectTitle *string               `json:"project_title,omitempty"`
	LeaderID     uuid.UUID             `json:"leader_id"`
	Finalized    bool                  `json:"finalized"`
	Members      []GroupMemberResponse `json:"members"`
	MemberCount  int                   `json:"member_count"`
	CreatedAt    time.Time             `json:"created_at"`
}

type GroupMemberResponse struct {
	StudentID   uuid.UUID  `json:"student_id"`
	StudentName string     `json:"student_name"`
	FromTeamID  *uuid.UUID `json:"from_team_id,omitempty"`
}

type SubmissionResponse struct {
	ID             uuid.UUID                `json:"id"`
	ProjectID      uuid.UUID                `json:"project_id"`
	ProjectGroupID uuid.UUID                `json:"project_group_id"`
	Content        *string                  `json:"content,omitempty"`
	Files          []SubmissionFileResponse `json:"files,omitempty"`
	IsSubmitted    bool                     `json:"is_submitted"`
	IsLate         bool                     `json:"is_late"`
	SubmittedAt    *time.Time               `json:"submitted_at,omitempty"`
	CreatedAt      time.Time                `json:"created_at"`
}

type SubmissionFileResponse struct {
	ID           uuid.UUID `json:"id"`
	StoredFileID uuid.UUID `json:"stored_file_id"`
	DisplayName  string    `json:"display_name"`
	OrderIndex   int       `json:"order_index"`
}

type SubmissionTeacherResponse struct {
	SubmissionResponse
	Grades []GradeResponse `json:"grades,omitempty"`
}

type GradeResponse struct {
	StudentID   uuid.UUID  `json:"student_id"`
	StudentName string     `json:"student_name"`
	Score       *float64   `json:"score"`
	Feedback    *string    `json:"feedback,omitempty"`
	GradedAt    *time.Time `json:"graded_at,omitempty"`
}

type MyGradeResponse struct {
	Score    *float64   `json:"score"`
	Feedback *string    `json:"feedback,omitempty"`
	GradedAt *time.Time `json:"graded_at,omitempty"`
	IsPublic bool       `json:"is_public"`
}

func ToProjectResponse(p *Project, now time.Time) ProjectResponse {
	return ProjectResponse{
		ID:                   p.ID,
		OfferingID:           p.OfferingID,
		Title:                p.Title,
		Body:                 p.Body,
		Deadline:             p.Deadline,
		MaxScore:             p.MaxScore,
		MinMembers:           p.MinMembers,
		MaxMembers:           p.MaxMembers,
		MergeTarget:          p.MergeTarget,
		RegistrationDeadline: p.RegistrationDeadline,
		Visibility:           p.Visibility,
		AllowLate:            p.AllowLate,
		PublishAt:            p.PublishAt,
		ScoresPublic:         p.ScoresPublic,
		IsPublished:          IsPublished(p.PublishAt, now),
		IsRegistrationOpen:   !IsRegistrationClosed(p.RegistrationDeadline, now),
		IsDeadlinePassed:     IsDeadlinePassed(p.Deadline, now),
		CreatedAt:            p.CreatedAt,
	}
}

func ToProjectsResponse(projects []Project, now time.Time) []ProjectResponse {
	result := make([]ProjectResponse, len(projects))
	for i, p := range projects {
		result[i] = ToProjectResponse(&p, now)
	}
	return result
}

func ToAttachmentResponse(a *ProjectAttachment) AttachmentResponse {
	return AttachmentResponse{
		ID:           a.ID,
		StoredFileID: a.StoredFileID,
		DisplayName:  a.DisplayName,
		OrderIndex:   a.OrderIndex,
		CreatedAt:    a.CreatedAt,
	}
}

func ToAttachmentsResponse(attachments []ProjectAttachment) []AttachmentResponse {
	result := make([]AttachmentResponse, len(attachments))
	for i, a := range attachments {
		result[i] = ToAttachmentResponse(&a)
	}
	return result
}

func ToRegistrationResponse(r *RegistrationWithTeam) RegistrationResponse {
	return RegistrationResponse{
		ID:           r.ID,
		ProjectID:    r.ProjectID,
		TeamID:       r.TeamID,
		TeamName:     r.TeamName,
		LeaderID:     r.LeaderID,
		LeaderName:   r.LeaderName,
		MemberCount:  r.MemberCount,
		ProjectTitle: r.ProjectTitle,
		RegisteredAt: r.RegisteredAt,
	}
}

func ToRegistrationsResponse(registrations []RegistrationWithTeam) []RegistrationResponse {
	result := make([]RegistrationResponse, len(registrations))
	for i, r := range registrations {
		result[i] = ToRegistrationResponse(&r)
	}
	return result
}

func ToProjectGroupResponse(g *ProjectGroupWithMembers) ProjectGroupResponse {
	members := make([]GroupMemberResponse, len(g.Members))
	for i, m := range g.Members {
		members[i] = GroupMemberResponse(m)
	}

	return ProjectGroupResponse{
		ID:           g.ID,
		ProjectID:    g.ProjectID,
		Name:         g.Name,
		ProjectTitle: g.ProjectTitle,
		LeaderID:     g.LeaderID,
		Finalized:    g.Finalized,
		Members:      members,
		MemberCount:  g.MemberCount,
		CreatedAt:    g.CreatedAt,
	}
}

func ToProjectGroupsResponse(groups []ProjectGroupWithMembers) []ProjectGroupResponse {
	result := make([]ProjectGroupResponse, len(groups))
	for i, g := range groups {
		result[i] = ToProjectGroupResponse(&g)
	}
	return result
}

func ToSubmissionResponse(s *Submission, files []SubmissionFile, deadline time.Time) SubmissionResponse {
	filesResp := make([]SubmissionFileResponse, len(files))
	for i, f := range files {
		filesResp[i] = SubmissionFileResponse{
			ID:           f.ID,
			StoredFileID: f.StoredFileID,
			DisplayName:  f.DisplayName,
			OrderIndex:   f.OrderIndex,
		}
	}

	isLate := false
	if s.SubmittedAt != nil {
		isLate = IsLateSubmission(deadline, *s.SubmittedAt)
	}

	return SubmissionResponse{
		ID:             s.ID,
		ProjectID:      s.ProjectID,
		ProjectGroupID: s.ProjectGroupID,
		Content:        s.Content,
		Files:          filesResp,
		IsSubmitted:    s.SubmittedAt != nil,
		IsLate:         isLate,
		SubmittedAt:    s.SubmittedAt,
		CreatedAt:      s.CreatedAt,
	}
}

func ToSubmissionTeacherResponse(s *Submission, files []SubmissionFile, grades []GradeWithStudent, deadline time.Time) SubmissionTeacherResponse {
	gradesResp := make([]GradeResponse, len(grades))
	for i, g := range grades {
		gradesResp[i] = GradeResponse{
			StudentID:   g.StudentID,
			StudentName: g.StudentName,
			Score:       g.Score,
			Feedback:    g.Feedback,
			GradedAt:    g.GradedAt,
		}
	}

	return SubmissionTeacherResponse{
		SubmissionResponse: ToSubmissionResponse(s, files, deadline),
		Grades:             gradesResp,
	}
}

func ToMyGradeResponse(g *Grade, scoresPublic bool) MyGradeResponse {
	resp := MyGradeResponse{
		IsPublic: scoresPublic,
	}
	if scoresPublic && g != nil {
		resp.Score = g.Score
		resp.Feedback = g.Feedback
		resp.GradedAt = g.GradedAt
	}
	return resp
}

func ToProjectUpdates(req UpdateProjectRequest) ProjectUpdates {
	return ProjectUpdates(req)
}

func ToFileInputs(files []FileInputRequest) []FileInput {
	result := make([]FileInput, len(files))
	for i, f := range files {
		result[i] = FileInput(f)
	}
	return result
}
