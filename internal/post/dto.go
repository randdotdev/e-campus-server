package post

import (
	"time"

	"github.com/google/uuid"
)

type CreatePostRequest struct {
	ScopeType string     `json:"scope_type" binding:"required,oneof=university college department program course"`
	ScopeID   *uuid.UUID `json:"scope_id"`
	Body      string     `json:"body" binding:"required,min=1,max=10000"`
	PublishAt *time.Time `json:"publish_at"`
	ExpiresAt *time.Time `json:"expires_at"`
}

type CreateCommentRequest struct {
	Body string `json:"body" binding:"required,min=1,max=5000"`
}

type UpdatePostRequest struct {
	Body      *string    `json:"body" binding:"omitempty,min=1,max=10000"`
	PublishAt *time.Time `json:"publish_at"`
	ExpiresAt *time.Time `json:"expires_at"`
}

type AddAttachmentRequest struct {
	StoredFileID uuid.UUID `json:"stored_file_id" binding:"required"`
	DisplayName  string    `json:"display_name" binding:"required,max=255"`
	FileType     string    `json:"file_type" binding:"required,oneof=image document voice video"`
	SizeBytes    int64     `json:"size_bytes" binding:"required,min=1"`
	OrderIndex   int       `json:"order_index"`
}

type PinPostRequest struct {
	Pinned bool `json:"pinned"`
}

type PostResponse struct {
	ID              uuid.UUID            `json:"id"`
	ScopeType       string               `json:"scope_type"`
	ScopeID         *uuid.UUID           `json:"scope_id,omitempty"`
	ParentID        *uuid.UUID           `json:"parent_id,omitempty"`
	RootID          *uuid.UUID           `json:"root_id,omitempty"`
	Body            string               `json:"body"`
	IsPinned        bool                 `json:"is_pinned"`
	PublishAt       *time.Time           `json:"publish_at,omitempty"`
	ExpiresAt       *time.Time           `json:"expires_at,omitempty"`
	Status          string               `json:"status"`
	AuthorID        uuid.UUID            `json:"author_id"`
	AuthorName      string               `json:"author_name"`
	AuthorNameLocal *string              `json:"author_name_local,omitempty"`
	AuthorAvatar    *string              `json:"author_avatar,omitempty"`
	AuthorRoleTitle *string              `json:"author_role_title,omitempty"`
	LikeCount       int                  `json:"like_count"`
	CommentCount    int                  `json:"comment_count"`
	IsLiked         bool                 `json:"is_liked"`
	Attachments     []AttachmentResponse `json:"attachments,omitempty"`
	Mentions        []MentionResponse    `json:"mentions,omitempty"`
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

type MentionResponse struct {
	UserID   uuid.UUID `json:"user_id"`
	Username string    `json:"username"`
	FullName string    `json:"full_name"`
}

func ToPostResponse(p *PostWithAuthor, attachments []PostAttachment, mentions []MentionedUser, isLiked bool, now time.Time) PostResponse {
	return PostResponse{
		ID:              p.ID,
		ScopeType:       p.ScopeType,
		ScopeID:         p.ScopeID,
		ParentID:        p.ParentID,
		RootID:          p.RootID,
		Body:            p.Body,
		IsPinned:        p.IsPinned,
		PublishAt:       p.PublishAt,
		ExpiresAt:       p.ExpiresAt,
		Status:          GetStatus(&p.Post, now),
		AuthorID:        p.AuthorID,
		AuthorName:      p.AuthorName,
		AuthorNameLocal: p.AuthorNameLocal,
		AuthorAvatar:    p.AuthorAvatar,
		AuthorRoleTitle: p.AuthorRoleTitle,
		LikeCount:       p.LikeCount,
		CommentCount:    p.CommentCount,
		IsLiked:         isLiked,
		Attachments:     ToAttachmentResponses(attachments),
		Mentions:        ToMentionResponses(mentions),
		CreatedAt:       p.CreatedAt,
		UpdatedAt:       p.UpdatedAt,
	}
}

func ToPostResponses(posts []PostWithAuthor, attachmentsMap map[uuid.UUID][]PostAttachment, mentionsMap map[uuid.UUID][]MentionedUser, likesMap map[uuid.UUID]bool, now time.Time) []PostResponse {
	result := make([]PostResponse, len(posts))
	for i := range posts {
		result[i] = ToPostResponse(
			&posts[i],
			attachmentsMap[posts[i].ID],
			mentionsMap[posts[i].ID],
			likesMap[posts[i].ID],
			now,
		)
	}
	return result
}

func ToAttachmentResponse(a *PostAttachment) AttachmentResponse {
	return AttachmentResponse{
		ID:           a.ID,
		StoredFileID: a.StoredFileID,
		DisplayName:  a.DisplayName,
		FileType:     a.FileType,
		OrderIndex:   a.OrderIndex,
	}
}

func ToAttachmentResponses(attachments []PostAttachment) []AttachmentResponse {
	if len(attachments) == 0 {
		return nil
	}
	result := make([]AttachmentResponse, len(attachments))
	for i := range attachments {
		result[i] = ToAttachmentResponse(&attachments[i])
	}
	return result
}

func ToMentionResponse(m *MentionedUser) MentionResponse {
	return MentionResponse{
		UserID:   m.UserID,
		Username: m.Username,
		FullName: m.FullName,
	}
}

func ToMentionResponses(mentions []MentionedUser) []MentionResponse {
	if len(mentions) == 0 {
		return nil
	}
	result := make([]MentionResponse, len(mentions))
	for i := range mentions {
		result[i] = ToMentionResponse(&mentions[i])
	}
	return result
}
