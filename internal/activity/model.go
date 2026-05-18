package activity

import (
	"time"

	"github.com/google/uuid"
)

type Activity struct {
	ID            uuid.UUID  `db:"id"`
	PublisherType string     `db:"publisher_type"`
	PublisherID   *uuid.UUID `db:"publisher_id"`
	Type          string     `db:"type"`
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

type ActivityWithAuthor struct {
	Activity
	AuthorName      string  `db:"author_name"`
	AuthorNameLocal *string `db:"author_name_local"`
	AuthorAvatar    *string `db:"author_avatar"`
}

type ActivityAttachment struct {
	ID           uuid.UUID `db:"id"`
	ActivityID   uuid.UUID `db:"activity_id"`
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
	TypeNews         = "news"
	TypeAnnouncement = "announcement"
	TypeWebinar      = "webinar"
	TypeWorkshop     = "workshop"
	TypeConference   = "conference"
	TypeSymposium    = "symposium"
	TypeTrainingCourse = "training_course"
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
	MaxImageSize    = 10 * 1024 * 1024 // 10MB
	MaxVideoSize    = 50 * 1024 * 1024 // 50MB
	MaxDocumentSize = 20 * 1024 * 1024 // 20MB
)

const (
	LangEN    = "en"
	LangLocal = "local"
)
