package activity

import (
	"time"

	"github.com/google/uuid"
)

type CreateActivityRequest struct {
	PublisherType string     `json:"publisher_type" binding:"required,oneof=university college department"`
	PublisherID   *uuid.UUID `json:"publisher_id"`
	Type          string     `json:"type" binding:"required,oneof=news announcement webinar workshop conference symposium training_course"`
	TitleEN       string     `json:"title_en" binding:"required,min=1,max=255"`
	TitleLocal    *string    `json:"title_local" binding:"omitempty,max=255"`
	BodyEN        string     `json:"body_en" binding:"required,min=1"`
	BodyLocal     *string    `json:"body_local"`
	CoverImageID  *uuid.UUID `json:"cover_image_id"`
	PublishAt     *time.Time `json:"publish_at"`
	ExpiresAt     *time.Time `json:"expires_at"`
}

type UpdateActivityRequest struct {
	TitleEN      *string    `json:"title_en" binding:"omitempty,min=1,max=255"`
	TitleLocal   *string    `json:"title_local" binding:"omitempty,max=255"`
	BodyEN       *string    `json:"body_en" binding:"omitempty,min=1"`
	BodyLocal    *string    `json:"body_local"`
	Type         *string    `json:"type" binding:"omitempty,oneof=news announcement webinar workshop conference symposium training_course"`
	CoverImageID *uuid.UUID `json:"cover_image_id"`
	PublishAt    *time.Time `json:"publish_at"`
	ExpiresAt    *time.Time `json:"expires_at"`
}

type AddAttachmentRequest struct {
	StoredFileID uuid.UUID `json:"stored_file_id" binding:"required"`
	DisplayName  string    `json:"display_name" binding:"required,max=255"`
	FileType     string    `json:"file_type" binding:"required,oneof=image document video"`
	SizeBytes    int64     `json:"size_bytes" binding:"required,min=1"`
	OrderIndex   int       `json:"order_index"`
}

type PinActivityRequest struct {
	Pinned bool `json:"pinned"`
}

type ActivityResponse struct {
	ID              uuid.UUID            `json:"id"`
	PublisherType   string               `json:"publisher_type"`
	PublisherID     *uuid.UUID           `json:"publisher_id,omitempty"`
	Type            string               `json:"type"`
	TitleEN         string               `json:"title_en"`
	TitleLocal      *string              `json:"title_local,omitempty"`
	BodyEN          string               `json:"body_en"`
	BodyLocal       *string              `json:"body_local,omitempty"`
	CoverImageID    *uuid.UUID           `json:"cover_image_id,omitempty"`
	IsPinned        bool                 `json:"is_pinned"`
	PublishAt       *time.Time           `json:"publish_at,omitempty"`
	ExpiresAt       *time.Time           `json:"expires_at,omitempty"`
	Status          string               `json:"status"`
	AuthorID        uuid.UUID            `json:"author_id"`
	AuthorName      string               `json:"author_name"`
	AuthorNameLocal *string              `json:"author_name_local,omitempty"`
	AuthorAvatar    *string              `json:"author_avatar,omitempty"`
	Attachments     []AttachmentResponse `json:"attachments,omitempty"`
	CreatedAt       time.Time            `json:"created_at"`
	UpdatedAt       *time.Time           `json:"updated_at,omitempty"`
}

type AttachmentResponse struct {
	ID           uuid.UUID `json:"id"`
	StoredFileID uuid.UUID `json:"stored_file_id"`
	DisplayName  string    `json:"display_name"`
	FileType     string    `json:"file_type"`
	OrderIndex   int       `json:"order_index"`
}

type TranslationResponse struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

func ToActivityResponse(a *ActivityWithAuthor, attachments []ActivityAttachment, now time.Time) ActivityResponse {
	return ActivityResponse{
		ID:              a.ID,
		PublisherType:   a.PublisherType,
		PublisherID:     a.PublisherID,
		Type:            a.Type,
		TitleEN:         a.TitleEN,
		TitleLocal:      a.TitleLocal,
		BodyEN:          a.BodyEN,
		BodyLocal:       a.BodyLocal,
		CoverImageID:    a.CoverImageID,
		IsPinned:        a.IsPinned,
		PublishAt:       a.PublishAt,
		ExpiresAt:       a.ExpiresAt,
		Status:          GetStatus(&a.Activity, now),
		AuthorID:        a.AuthorID,
		AuthorName:      a.AuthorName,
		AuthorNameLocal: a.AuthorNameLocal,
		AuthorAvatar:    a.AuthorAvatar,
		Attachments:     ToAttachmentResponses(attachments),
		CreatedAt:       a.CreatedAt,
		UpdatedAt:       a.UpdatedAt,
	}
}

func ToActivityResponses(activities []ActivityWithAuthor, attachmentsMap map[uuid.UUID][]ActivityAttachment, now time.Time) []ActivityResponse {
	result := make([]ActivityResponse, len(activities))
	for i := range activities {
		result[i] = ToActivityResponse(&activities[i], attachmentsMap[activities[i].ID], now)
	}
	return result
}

func ToAttachmentResponse(a *ActivityAttachment) AttachmentResponse {
	return AttachmentResponse{
		ID:           a.ID,
		StoredFileID: a.StoredFileID,
		DisplayName:  a.DisplayName,
		FileType:     a.FileType,
		OrderIndex:   a.OrderIndex,
	}
}

func ToAttachmentResponses(attachments []ActivityAttachment) []AttachmentResponse {
	if len(attachments) == 0 {
		return nil
	}
	result := make([]AttachmentResponse, len(attachments))
	for i := range attachments {
		result[i] = ToAttachmentResponse(&attachments[i])
	}
	return result
}
