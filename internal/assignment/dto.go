package assignment

import (
	"time"

	"github.com/google/uuid"
)

// Request types

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
	Title     *string    `json:"title" binding:"omitempty,max=255"`
	Body      *string    `json:"body"`
	Type      *string    `json:"type" binding:"omitempty,oneof=theory practice"`
	Deadline  *time.Time `json:"deadline"`
	MaxScore  *float64   `json:"max_score" binding:"omitempty,gt=0"`
	AllowLate *bool      `json:"allow_late"`
	PublishAt *time.Time `json:"publish_at"`
}

type AddAttachmentRequest struct {
	StoredFileID uuid.UUID `json:"stored_file_id" binding:"required"`
	DisplayName  string    `json:"display_name" binding:"required,max=255"`
	OrderIndex   int       `json:"order_index"`
}

type CreateSubmissionRequest struct {
	Content *string         `json:"content"`
	Files   []FileInputDTO  `json:"files"`
}

type UpdateSubmissionRequest struct {
	Content *string         `json:"content"`
	Files   []FileInputDTO  `json:"files"`
}

type GradeRequest struct {
	Score    float64 `json:"score" binding:"min=0"`
	Feedback *string `json:"feedback"`
}

type FileInputDTO struct {
	StoredFileID uuid.UUID `json:"stored_file_id" binding:"required"`
	DisplayName  string    `json:"display_name" binding:"required,max=255"`
}

// Response types

type AssignmentResponse struct {
	ID           uuid.UUID  `json:"id"`
	OfferingID   uuid.UUID  `json:"offering_id"`
	Title        string     `json:"title"`
	Body         *string    `json:"body,omitempty"`
	Type         *string    `json:"type,omitempty"`
	Deadline     time.Time  `json:"deadline"`
	MaxScore     float64    `json:"max_score"`
	AllowLate    bool       `json:"allow_late"`
	IsPublished  bool       `json:"is_published"`
	ScoresPublic bool       `json:"scores_public"`
	CreatedAt    time.Time  `json:"created_at"`
}

type AssignmentWithAttachmentsResponse struct {
	AssignmentResponse
	Attachments []AttachmentResponse `json:"attachments"`
}

type AttachmentResponse struct {
	ID           uuid.UUID `json:"id"`
	StoredFileID uuid.UUID `json:"stored_file_id"`
	DisplayName  string    `json:"display_name"`
	OrderIndex   int       `json:"order_index"`
}

type SubmissionResponse struct {
	ID           uuid.UUID            `json:"id"`
	AssignmentID uuid.UUID            `json:"assignment_id"`
	StudentID    uuid.UUID            `json:"student_id"`
	Content      *string              `json:"content,omitempty"`
	Files        []SubmissionFileResponse `json:"files,omitempty"`
	Status       string               `json:"status"`
	IsLate       bool                 `json:"is_late"`
	SubmittedAt  *time.Time           `json:"submitted_at,omitempty"`
	CreatedAt    time.Time            `json:"created_at"`
	UpdatedAt    *time.Time           `json:"updated_at,omitempty"`
}

type SubmissionWithScoreResponse struct {
	SubmissionResponse
	Score    *float64 `json:"score,omitempty"`
	Feedback *string  `json:"feedback,omitempty"`
}

type SubmissionTeacherResponse struct {
	SubmissionResponse
	StudentName string     `json:"student_name"`
	Score       *float64   `json:"score,omitempty"`
	Feedback    *string    `json:"feedback,omitempty"`
	GradedBy    *uuid.UUID `json:"graded_by,omitempty"`
	GradedAt    *time.Time `json:"graded_at,omitempty"`
}

type SubmissionFileResponse struct {
	ID           uuid.UUID `json:"id"`
	StoredFileID uuid.UUID `json:"stored_file_id"`
	DisplayName  string    `json:"display_name"`
	OrderIndex   int       `json:"order_index"`
}

// Mappers

func ToAssignmentResponse(a *Assignment, now time.Time) AssignmentResponse {
	return AssignmentResponse{
		ID:           a.ID,
		OfferingID:   a.OfferingID,
		Title:        a.Title,
		Body:         a.Body,
		Type:         a.Type,
		Deadline:     a.Deadline,
		MaxScore:     a.MaxScore,
		AllowLate:    a.AllowLate,
		IsPublished:  IsPublished(a.PublishAt, now),
		ScoresPublic: a.ScoresPublic,
		CreatedAt:    a.CreatedAt,
	}
}

func ToAssignmentsResponse(assignments []Assignment, now time.Time) []AssignmentResponse {
	result := make([]AssignmentResponse, len(assignments))
	for i := range assignments {
		result[i] = ToAssignmentResponse(&assignments[i], now)
	}
	return result
}

func ToAssignmentWithAttachmentsResponse(a *Assignment, attachments []AssignmentAttachment, now time.Time) AssignmentWithAttachmentsResponse {
	return AssignmentWithAttachmentsResponse{
		AssignmentResponse: ToAssignmentResponse(a, now),
		Attachments:        ToAttachmentsResponse(attachments),
	}
}

func ToAttachmentResponse(a *AssignmentAttachment) AttachmentResponse {
	return AttachmentResponse{
		ID:           a.ID,
		StoredFileID: a.StoredFileID,
		DisplayName:  a.DisplayName,
		OrderIndex:   a.OrderIndex,
	}
}

func ToAttachmentsResponse(attachments []AssignmentAttachment) []AttachmentResponse {
	result := make([]AttachmentResponse, len(attachments))
	for i := range attachments {
		result[i] = ToAttachmentResponse(&attachments[i])
	}
	return result
}

func ToSubmissionResponse(s *Submission, files []SubmissionFile, deadline time.Time) SubmissionResponse {
	resp := SubmissionResponse{
		ID:           s.ID,
		AssignmentID: s.AssignmentID,
		StudentID:    s.StudentID,
		Content:      s.Content,
		Files:        ToSubmissionFilesResponse(files),
		Status:       ComputeStatus(s.SubmittedAt, s.GradedAt),
		SubmittedAt:  s.SubmittedAt,
		CreatedAt:    s.CreatedAt,
		UpdatedAt:    s.UpdatedAt,
	}
	if s.SubmittedAt != nil {
		resp.IsLate = IsLate(deadline, *s.SubmittedAt)
	}
	return resp
}

func ToSubmissionWithScoreResponse(s *Submission, files []SubmissionFile, deadline time.Time, scoresPublic bool) SubmissionWithScoreResponse {
	resp := SubmissionWithScoreResponse{
		SubmissionResponse: ToSubmissionResponse(s, files, deadline),
	}
	if scoresPublic {
		resp.Score = s.Score
		resp.Feedback = s.Feedback
	}
	return resp
}

func ToSubmissionTeacherResponse(s *SubmissionWithStudent, files []SubmissionFile, deadline time.Time) SubmissionTeacherResponse {
	return SubmissionTeacherResponse{
		SubmissionResponse: ToSubmissionResponse(&s.Submission, files, deadline),
		StudentName:        s.StudentName,
		Score:              s.Score,
		Feedback:           s.Feedback,
		GradedBy:           s.GradedBy,
		GradedAt:           s.GradedAt,
	}
}

func ToSubmissionFileResponse(f *SubmissionFile) SubmissionFileResponse {
	return SubmissionFileResponse{
		ID:           f.ID,
		StoredFileID: f.StoredFileID,
		DisplayName:  f.DisplayName,
		OrderIndex:   f.OrderIndex,
	}
}

func ToSubmissionFilesResponse(files []SubmissionFile) []SubmissionFileResponse {
	result := make([]SubmissionFileResponse, len(files))
	for i := range files {
		result[i] = ToSubmissionFileResponse(&files[i])
	}
	return result
}

func ToFileInputs(dtos []FileInputDTO) []FileInput {
	result := make([]FileInput, len(dtos))
	for i, d := range dtos {
		result[i] = FileInput(d)
	}
	return result
}

func ToAssignmentUpdates(req UpdateAssignmentRequest) AssignmentUpdates {
	return AssignmentUpdates(req)
}
