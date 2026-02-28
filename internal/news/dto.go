package news

import (
	"time"

	"github.com/google/uuid"
)

type CreateNewsRequest struct {
	PublisherType string     `json:"publisher_type" binding:"required,oneof=university college department"`
	PublisherID   *uuid.UUID `json:"publisher_id"`
	Category      string     `json:"category" binding:"required,oneof=announcement event achievement academic general"`
	TitleEN       string     `json:"title_en" binding:"required,min=1,max=255"`
	TitleLocal    *string    `json:"title_local" binding:"omitempty,max=255"`
	BodyEN        string     `json:"body_en" binding:"required,min=1"`
	BodyLocal     *string    `json:"body_local"`
	CoverImageID  *uuid.UUID `json:"cover_image_id"`
	PublishAt     *time.Time `json:"publish_at"`
	ExpiresAt     *time.Time `json:"expires_at"`
}

type UpdateNewsRequest struct {
	TitleEN      *string    `json:"title_en" binding:"omitempty,min=1,max=255"`
	TitleLocal   *string    `json:"title_local" binding:"omitempty,max=255"`
	BodyEN       *string    `json:"body_en" binding:"omitempty,min=1"`
	BodyLocal    *string    `json:"body_local"`
	Category     *string    `json:"category" binding:"omitempty,oneof=announcement event achievement academic general"`
	CoverImageID *uuid.UUID `json:"cover_image_id"`
	PublishAt    *time.Time `json:"publish_at"`
	ExpiresAt    *time.Time `json:"expires_at"`
}

type AddAttachmentRequest struct {
	StoredFileID uuid.UUID `json:"stored_file_id" binding:"required"`
	DisplayName  string    `json:"display_name" binding:"required,max=255"`
	FileType     string    `json:"file_type" binding:"required,oneof=image document video"`
	OrderIndex   int       `json:"order_index"`
}

type PinNewsRequest struct {
	Pinned bool `json:"pinned"`
}

type NewsResponse struct {
	ID              uuid.UUID            `json:"id"`
	PublisherType   string               `json:"publisher_type"`
	PublisherID     *uuid.UUID           `json:"publisher_id,omitempty"`
	Category        string               `json:"category"`
	Title           string               `json:"title"`
	Body            string               `json:"body"`
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

func ToNewsResponse(n *NewsWithAuthor, attachments []NewsAttachment, prefLang, defaultLang string, now time.Time) NewsResponse {
	return NewsResponse{
		ID:              n.ID,
		PublisherType:   n.PublisherType,
		PublisherID:     n.PublisherID,
		Category:        n.Category,
		Title:           ResolveTitle(&n.News, prefLang, defaultLang),
		Body:            ResolveBody(&n.News, prefLang, defaultLang),
		CoverImageID:    n.CoverImageID,
		IsPinned:        n.IsPinned,
		PublishAt:       n.PublishAt,
		ExpiresAt:       n.ExpiresAt,
		Status:          GetStatus(&n.News, now),
		AuthorID:        n.AuthorID,
		AuthorName:      n.AuthorName,
		AuthorNameLocal: n.AuthorNameLocal,
		AuthorAvatar:    n.AuthorAvatar,
		Attachments:     ToAttachmentResponses(attachments),
		CreatedAt:       n.CreatedAt,
		UpdatedAt:       n.UpdatedAt,
	}
}

func ToNewsResponses(news []NewsWithAuthor, attachmentsMap map[uuid.UUID][]NewsAttachment, prefLang, defaultLang string, now time.Time) []NewsResponse {
	result := make([]NewsResponse, len(news))
	for i := range news {
		result[i] = ToNewsResponse(&news[i], attachmentsMap[news[i].ID], prefLang, defaultLang, now)
	}
	return result
}

func ToAttachmentResponse(a *NewsAttachment) AttachmentResponse {
	return AttachmentResponse{
		ID:           a.ID,
		StoredFileID: a.StoredFileID,
		DisplayName:  a.DisplayName,
		FileType:     a.FileType,
		OrderIndex:   a.OrderIndex,
	}
}

func ToAttachmentResponses(attachments []NewsAttachment) []AttachmentResponse {
	if len(attachments) == 0 {
		return nil
	}
	result := make([]AttachmentResponse, len(attachments))
	for i := range attachments {
		result[i] = ToAttachmentResponse(&attachments[i])
	}
	return result
}
