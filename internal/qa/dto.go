package qa

import (
	"time"

	"github.com/google/uuid"
)

type AskQuestionRequest struct {
	Title       string `json:"title" binding:"required,min=1,max=255"`
	Body        string `json:"body" binding:"required,min=1"`
	IsAnonymous bool   `json:"is_anonymous"`
}

type CreateFAQRequest struct {
	Title        string `json:"title" binding:"required,min=1,max=255"`
	QuestionBody string `json:"question_body" binding:"required,min=1"`
	AnswerBody   string `json:"answer_body" binding:"required,min=1"`
}

type UpdateQuestionRequest struct {
	Title *string `json:"title" binding:"omitempty,min=1,max=255"`
	Body  *string `json:"body" binding:"omitempty,min=1"`
}

type AnswerQuestionRequest struct {
	Body         string  `json:"body" binding:"required,min=1"`
	QuestionEdit *string `json:"question_edit"`
}

type UpdateAnswerRequest struct {
	Body string `json:"body" binding:"required,min=1"`
}

type RejectQuestionRequest struct {
	Reason string `json:"reason" binding:"required,min=1"`
}

type QuestionResponse struct {
	ID              uuid.UUID            `json:"id"`
	OfferingID      uuid.UUID            `json:"offering_id"`
	Title           string               `json:"title"`
	Body            string               `json:"body"`
	IsAnonymous     bool                 `json:"is_anonymous"`
	IsFAQ           bool                 `json:"is_faq"`
	Status          string               `json:"status"`
	Rejection       *RejectionResponse   `json:"rejection,omitempty"`
	AuthorID        *uuid.UUID           `json:"author_id,omitempty"`
	AuthorName      *string              `json:"author_name,omitempty"`
	AuthorNameLocal *string              `json:"author_name_local,omitempty"`
	EditedBy        *uuid.UUID           `json:"edited_by,omitempty"`
	Answer          *AnswerResponse      `json:"answer,omitempty"`
	Attachments     []AttachmentResponse `json:"attachments,omitempty"`
	CreatedAt       time.Time            `json:"created_at"`
	UpdatedAt       *time.Time           `json:"updated_at,omitempty"`
}

type RejectionResponse struct {
	Reason              string    `json:"reason"`
	RejectedBy          uuid.UUID `json:"rejected_by"`
	RejectedByName      string    `json:"rejected_by_name"`
	RejectedByNameLocal *string   `json:"rejected_by_name_local,omitempty"`
	RejectedAt          time.Time `json:"rejected_at"`
}

type AnswerResponse struct {
	ID              uuid.UUID            `json:"id"`
	Body            string               `json:"body"`
	AuthorID        uuid.UUID            `json:"author_id"`
	AuthorName      string               `json:"author_name"`
	AuthorNameLocal *string              `json:"author_name_local,omitempty"`
	Attachments     []AttachmentResponse `json:"attachments,omitempty"`
	CreatedAt       time.Time            `json:"created_at"`
	UpdatedAt       *time.Time           `json:"updated_at,omitempty"`
}

type AttachmentResponse struct {
	ID       uuid.UUID `json:"id"`
	FilePath string    `json:"file_path"`
	FileName string    `json:"file_name"`
	FileSize int       `json:"file_size"`
	MimeType string    `json:"mime_type"`
}

func ToQuestionResponse(q *QuestionWithAuthor, answer *AnswerWithAuthor, rejection *QuestionRejectionWithUser, qAttachments []QuestionAttachment, aAttachments []AnswerAttachment, isTeacher bool) QuestionResponse {
	resp := QuestionResponse{
		ID:          q.ID,
		OfferingID:  q.OfferingID,
		Title:       q.Title,
		Body:        q.Body,
		IsAnonymous: q.IsAnonymous,
		IsFAQ:       q.IsFAQ,
		Status:      q.Status,
		EditedBy:    q.EditedBy,
		Attachments: ToQuestionAttachmentResponses(qAttachments),
		CreatedAt:   q.CreatedAt,
		UpdatedAt:   q.UpdatedAt,
	}

	if rejection != nil {
		resp.Rejection = ToRejectionResponse(rejection)
	}

	if !q.IsAnonymous || isTeacher {
		resp.AuthorID = &q.CreatedBy
		resp.AuthorName = &q.AuthorName
		resp.AuthorNameLocal = q.AuthorNameLocal
	}

	if answer != nil {
		resp.Answer = ToAnswerResponse(answer, aAttachments)
	}

	return resp
}

func ToRejectionResponse(r *QuestionRejectionWithUser) *RejectionResponse {
	return &RejectionResponse{
		Reason:              r.Reason,
		RejectedBy:          r.RejectedBy,
		RejectedByName:      r.RejectedByName,
		RejectedByNameLocal: r.RejectedByNameLocal,
		RejectedAt:          r.RejectedAt,
	}
}

func ToQuestionListResponse(q *QuestionWithAuthor, isTeacher bool) QuestionResponse {
	resp := QuestionResponse{
		ID:          q.ID,
		OfferingID:  q.OfferingID,
		Title:       q.Title,
		Body:        q.Body,
		IsAnonymous: q.IsAnonymous,
		IsFAQ:       q.IsFAQ,
		Status:      q.Status,
		CreatedAt:   q.CreatedAt,
		UpdatedAt:   q.UpdatedAt,
	}

	if !q.IsAnonymous || isTeacher {
		resp.AuthorID = &q.CreatedBy
		resp.AuthorName = &q.AuthorName
		resp.AuthorNameLocal = q.AuthorNameLocal
	}

	return resp
}

func ToQuestionListResponses(questions []QuestionWithAuthor, isTeacher bool) []QuestionResponse {
	result := make([]QuestionResponse, len(questions))
	for i := range questions {
		result[i] = ToQuestionListResponse(&questions[i], isTeacher)
	}
	return result
}

func ToAnswerResponse(a *AnswerWithAuthor, attachments []AnswerAttachment) *AnswerResponse {
	return &AnswerResponse{
		ID:              a.ID,
		Body:            a.Body,
		AuthorID:        a.CreatedBy,
		AuthorName:      a.AuthorName,
		AuthorNameLocal: a.AuthorNameLocal,
		Attachments:     ToAnswerAttachmentResponses(attachments),
		CreatedAt:       a.CreatedAt,
		UpdatedAt:       a.UpdatedAt,
	}
}

func ToQuestionAttachmentResponse(a *QuestionAttachment) AttachmentResponse {
	return AttachmentResponse{
		ID:       a.ID,
		FilePath: a.FilePath,
		FileName: a.FileName,
		FileSize: a.FileSize,
		MimeType: a.MimeType,
	}
}

func ToQuestionAttachmentResponses(attachments []QuestionAttachment) []AttachmentResponse {
	if len(attachments) == 0 {
		return nil
	}
	result := make([]AttachmentResponse, len(attachments))
	for i := range attachments {
		result[i] = ToQuestionAttachmentResponse(&attachments[i])
	}
	return result
}

func ToAnswerAttachmentResponse(a *AnswerAttachment) AttachmentResponse {
	return AttachmentResponse{
		ID:       a.ID,
		FilePath: a.FilePath,
		FileName: a.FileName,
		FileSize: a.FileSize,
		MimeType: a.MimeType,
	}
}

func ToAnswerAttachmentResponses(attachments []AnswerAttachment) []AttachmentResponse {
	if len(attachments) == 0 {
		return nil
	}
	result := make([]AttachmentResponse, len(attachments))
	for i := range attachments {
		result[i] = ToAnswerAttachmentResponse(&attachments[i])
	}
	return result
}
