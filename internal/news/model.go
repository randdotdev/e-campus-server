// Package news handles university news and announcements.
package news

import (
	"time"

	"github.com/google/uuid"
)

type News struct {
	ID            uuid.UUID  `db:"id"`
	PublisherType string     `db:"publisher_type"`
	PublisherID   *uuid.UUID `db:"publisher_id"`
	Category      string     `db:"category"`
	TitleEN       string     `db:"title_en"`
	TitleLocal    *string    `db:"title_local"`
	BodyEN        string     `db:"body_en"`
	BodyLocal     *string    `db:"body_local"`
	CoverImageID  *uuid.UUID `db:"cover_image_id"`
	AuthorID      uuid.UUID  `db:"author_id"`
	IsPinned      bool       `db:"is_pinned"`
	PublishAt     *time.Time `db:"publish_at"`
	ExpiresAt     *time.Time `db:"expires_at"`
	CreatedAt     time.Time  `db:"created_at"`
	UpdatedAt     *time.Time `db:"updated_at"`
	DeletedAt     *time.Time `db:"deleted_at"`
}

type NewsWithAuthor struct {
	News
	AuthorName      string  `db:"author_name"`
	AuthorNameLocal *string `db:"author_name_local"`
	AuthorAvatar    *string `db:"author_avatar"`
}

type NewsAttachment struct {
	ID           uuid.UUID `db:"id"`
	NewsID       uuid.UUID `db:"news_id"`
	StoredFileID uuid.UUID `db:"stored_file_id"`
	DisplayName  string    `db:"display_name"`
	FileType     string    `db:"file_type"`
	OrderIndex   int       `db:"order_index"`
}

const (
	PublisherUniversity = "university"
	PublisherCollege    = "college"
	PublisherDepartment = "department"
)

const (
	CategoryAnnouncement = "announcement"
	CategoryEvent        = "event"
	CategoryAchievement  = "achievement"
	CategoryAcademic     = "academic"
	CategoryGeneral      = "general"
)

const (
	StatusScheduled = "scheduled"
	StatusPublished = "published"
	StatusExpired   = "expired"
)

const (
	FileTypeImage    = "image"
	FileTypeDocument = "document"
	FileTypeVideo    = "video"
)

const (
	LangEN    = "en"
	LangLocal = "local"
)
