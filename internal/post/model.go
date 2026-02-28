// Package post handles internal communication for university members.
package post

import (
	"time"

	"github.com/google/uuid"
)

type Post struct {
	ID           uuid.UUID  `db:"id"`
	ScopeType    string     `db:"scope_type"`
	ScopeID      *uuid.UUID `db:"scope_id"`
	ParentID     *uuid.UUID `db:"parent_id"`
	RootID       *uuid.UUID `db:"root_id"`
	Body         string     `db:"body"`
	IsPinned     bool       `db:"is_pinned"`
	PublishAt    *time.Time `db:"publish_at"`
	ExpiresAt    *time.Time `db:"expires_at"`
	AuthorID     uuid.UUID  `db:"author_id"`
	LikeCount    int        `db:"like_count"`
	CommentCount int        `db:"comment_count"`
	CreatedAt    time.Time  `db:"created_at"`
	UpdatedAt    *time.Time `db:"updated_at"`
	DeletedAt    *time.Time `db:"deleted_at"`
}

type PostWithAuthor struct {
	Post
	AuthorName     string  `db:"author_name"`
	AuthorNameLocal   *string `db:"author_name_local"`
	AuthorAvatar   *string `db:"author_avatar"`
	AuthorRoleTitle *string `db:"author_role_title"`
}

type PostAttachment struct {
	ID           uuid.UUID `db:"id"`
	PostID       uuid.UUID `db:"post_id"`
	StoredFileID uuid.UUID `db:"stored_file_id"`
	DisplayName  string    `db:"display_name"`
	FileType     string    `db:"file_type"`
	OrderIndex   int       `db:"order_index"`
}

type PostLike struct {
	PostID    uuid.UUID `db:"post_id"`
	UserID    uuid.UUID `db:"user_id"`
	CreatedAt time.Time `db:"created_at"`
}

type PostMention struct {
	PostID uuid.UUID `db:"post_id"`
	UserID uuid.UUID `db:"user_id"`
}

type MentionedUser struct {
	UserID   uuid.UUID `db:"user_id"`
	Username string    `db:"username"`
	FullName string    `db:"full_name"`
}

// Scope types
const (
	ScopeUniversity = "university"
	ScopeCollege    = "college"
	ScopeDepartment = "department"
	ScopeProgram    = "program"
)

// Post status
const (
	StatusScheduled = "scheduled"
	StatusPublished = "published"
	StatusExpired   = "expired"
)

// File types for attachments
const (
	FileTypeImage    = "image"
	FileTypeDocument = "document"
	FileTypeVoice    = "voice"
	FileTypeVideo    = "video"
)

// File size limits in bytes
const (
	MaxImageSize    = 10 * 1024 * 1024  // 10MB
	MaxVideoSize    = 50 * 1024 * 1024  // 50MB
	MaxVoiceSize    = 10 * 1024 * 1024  // 10MB
	MaxDocumentSize = 20 * 1024 * 1024  // 20MB
)
