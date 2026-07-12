package announcements

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/shared/pagination"
)

// ── Value objects ──────────────────────────────────────────────────────────

// PublisherType is who publishes an activity. The same closed set is a CHECK
// constraint on activities.publisher_type.
type PublisherType string

// Publisher types.
const (
	PublisherUniversity PublisherType = "university"
	PublisherCollege    PublisherType = "college"
	PublisherDepartment PublisherType = "department"
)

// ActivityType is the kind of institutional activity. The same closed set is
// a CHECK constraint on activities.type.
type ActivityType string

// Activity types.
const (
	TypeNews           ActivityType = "news"
	TypeAnnouncement   ActivityType = "announcement"
	TypeWebinar        ActivityType = "webinar"
	TypeWorkshop       ActivityType = "workshop"
	TypeConference     ActivityType = "conference"
	TypeSymposium      ActivityType = "symposium"
	TypeTrainingCourse ActivityType = "training_course"
)

// Lang selects which language variant of an activity to read.
type Lang string

// Languages.
const (
	LangEN    Lang = "en"
	LangLocal Lang = "local"
)

// ── Entities ───────────────────────────────────────────────────────────────

// Activity is one institutional feed entry: news, an announcement, or an
// event, bilingual and soft-deletable.
type Activity struct {
	ID            uuid.UUID     `db:"id"`
	PublisherType PublisherType `db:"publisher_type"`
	PublisherID   *uuid.UUID    `db:"publisher_id"`
	Type          ActivityType  `db:"type"`
	TitleEN       string        `db:"title_en"`
	TitleLocal    *string       `db:"title_local"`
	BodyEN        string        `db:"body_en"`
	BodyLocal     *string       `db:"body_local"`
	CoverImageID  *uuid.UUID    `db:"cover_image_id"`
	AuthorID      uuid.UUID     `db:"author_id"`
	IsPinned      bool          `db:"is_pinned"`
	PublishAt     *time.Time    `db:"publish_at"`
	ExpiresAt     *time.Time    `db:"expires_at"`
	Version       int64         `db:"version"`
	CreatedAt     time.Time     `db:"created_at"`
	UpdatedAt     *time.Time    `db:"updated_at"`
	DeletedAt     *time.Time    `db:"deleted_at"`
}

// ActivityWithAuthor is the activity joined with the author's display
// columns (activities ⋈ users, the published identity columns).
type ActivityWithAuthor struct {
	Activity
	AuthorName      string  `db:"author_name"`
	AuthorNameLocal *string `db:"author_name_local"`
	AuthorAvatar    *string `db:"author_avatar"`
}

// ActivityAttachment is one file attached to an activity.
type ActivityAttachment struct {
	ID          uuid.UUID `db:"id"`
	ActivityID  uuid.UUID `db:"activity_id"`
	InodeID     uuid.UUID `db:"inode_id"`
	DisplayName string    `db:"display_name"`
	FileType    string    `db:"file_type"`
	OrderIndex  int       `db:"order_index"`
}

// ── Rules (behaviour on the entity) ────────────────────────────────────────

// Visible reports whether the activity is live for regular readers.
func (a *Activity) Visible(now time.Time) bool {
	return visible(a.DeletedAt, a.PublishAt, a.ExpiresAt, now)
}

// CanView reports whether the reader may see the activity; admins see
// scheduled and expired ones.
func (a *Activity) CanView(revealHidden bool, now time.Time) bool {
	return canView(a.DeletedAt, a.PublishAt, a.ExpiresAt, revealHidden, now)
}

// Status returns the activity's publish lifecycle state.
func (a *Activity) Status(now time.Time) Status { return statusOf(a.PublishAt, a.ExpiresAt, now) }

// ValidPublisherType reports whether pt is a known publisher type.
func ValidPublisherType(pt PublisherType) bool {
	switch pt {
	case PublisherUniversity, PublisherCollege, PublisherDepartment:
		return true
	}
	return false
}

// ValidPublisherID reports whether the publisher ID's presence matches
// the publisher type: the university carries none, colleges and departments
// require one.
func ValidPublisherID(pt PublisherType, publisherID *uuid.UUID) bool {
	if pt == PublisherUniversity {
		return publisherID == nil
	}
	return publisherID != nil
}

// ValidActivityType reports whether t is a known activity type.
func ValidActivityType(t ActivityType) bool {
	switch t {
	case TypeNews, TypeAnnouncement, TypeWebinar, TypeWorkshop, TypeConference, TypeSymposium, TypeTrainingCourse:
		return true
	}
	return false
}

// ValidLanguage reports whether lang is a known language selector.
func ValidLanguage(lang Lang) bool { return lang == LangEN || lang == LangLocal }

// ResolveTitle returns the activity title in the preferred language, falling
// back to whichever variant exists.
func ResolveTitle(a *Activity, preferred Lang) string {
	if preferred == LangLocal && a.TitleLocal != nil && *a.TitleLocal != "" {
		return *a.TitleLocal
	}
	if a.TitleEN != "" {
		return a.TitleEN
	}
	if a.TitleLocal != nil {
		return *a.TitleLocal
	}
	return ""
}

// ResolveBody returns the activity body in the preferred language, falling
// back to whichever variant exists.
func ResolveBody(a *Activity, preferred Lang) string {
	if preferred == LangLocal && a.BodyLocal != nil && *a.BodyLocal != "" {
		return *a.BodyLocal
	}
	if a.BodyEN != "" {
		return a.BodyEN
	}
	if a.BodyLocal != nil {
		return *a.BodyLocal
	}
	return ""
}

// ResolveTranslation returns the exact language variant, reporting ok=false
// when that variant is incomplete.
func ResolveTranslation(a *Activity, lang Lang) (title, body string, ok bool) {
	if lang == LangLocal {
		if a.TitleLocal == nil || a.BodyLocal == nil {
			return "", "", false
		}
		return *a.TitleLocal, *a.BodyLocal, true
	}
	if a.TitleEN == "" || a.BodyEN == "" {
		return "", "", false
	}
	return a.TitleEN, a.BodyEN, true
}

// BuildActivity constructs a new activity from its input.
func BuildActivity(in CreateActivityInput) *Activity {
	return &Activity{
		ID:            uuid.New(),
		PublisherType: in.PublisherType,
		PublisherID:   in.PublisherID,
		Type:          in.Type,
		TitleEN:       in.TitleEN,
		TitleLocal:    in.TitleLocal,
		BodyEN:        in.BodyEN,
		BodyLocal:     in.BodyLocal,
		AuthorID:      in.AuthorID,
		PublishAt:     in.PublishAt,
		ExpiresAt:     in.ExpiresAt,
		CreatedAt:     time.Now(),
	}
}

// BuildActivityAttachment constructs an attachment row for an activity.
func BuildActivityAttachment(activityID, inodeID uuid.UUID, displayName, fileType string, orderIndex int) *ActivityAttachment {
	return &ActivityAttachment{
		ID:          uuid.New(),
		ActivityID:  activityID,
		InodeID:     inodeID,
		DisplayName: displayName,
		FileType:    fileType,
		OrderIndex:  orderIndex,
	}
}

// ── Ports ──────────────────────────────────────────────────────────────────

// ActivityRepository persists activities and their attachments. Get* methods
// return nil (no error) when the row does not exist. UpdateActivity is an
// optimistic compare-and-swap keyed on expectedVersion, returning the new
// version; a version mismatch is ErrConflict.
type ActivityRepository interface {
	CreateActivity(ctx context.Context, a *Activity) error
	GetActivityByID(ctx context.Context, id uuid.UUID) (*Activity, error)
	GetActivityByIDWithAuthor(ctx context.Context, id uuid.UUID) (*ActivityWithAuthor, error)
	UpdateActivity(ctx context.Context, a *Activity, expectedVersion int64) (int64, error)
	SoftDeleteActivity(ctx context.Context, id uuid.UUID, deletedAt time.Time) error
	ListActivitiesByPublisher(ctx context.Context, pt PublisherType, publisherID *uuid.UUID, activityType ActivityType, revealHidden bool, params pagination.PageParams) ([]ActivityWithAuthor, bool, error)
	CreateAttachment(ctx context.Context, a *ActivityAttachment) error
	DeleteAttachment(ctx context.Context, id uuid.UUID) error
	GetAttachmentByID(ctx context.Context, id uuid.UUID) (*ActivityAttachment, error)
	ListAttachmentsByActivityID(ctx context.Context, activityID uuid.UUID) ([]ActivityAttachment, error)
	ListAttachmentsByActivityIDs(ctx context.Context, activityIDs []uuid.UUID) (map[uuid.UUID][]ActivityAttachment, error)
}

// PublisherChecker verifies a college/department publisher exists.
type PublisherChecker interface {
	PublisherExists(ctx context.Context, pt PublisherType, publisherID uuid.UUID) (bool, error)
}

// DefaultLanguageProvider yields the institution's configured default language.
// Implemented by management/settings and injected by the composition root.
type DefaultLanguageProvider interface {
	GetDefaultLanguage(ctx context.Context) (string, error)
}

// ── Service input types ────────────────────────────────────────────────────

// CreateActivityInput is the content of a new activity.
type CreateActivityInput struct {
	AuthorID      uuid.UUID
	PublisherType PublisherType
	PublisherID   *uuid.UUID
	Type          ActivityType
	TitleEN       string
	TitleLocal    *string
	BodyEN        string
	BodyLocal     *string
	// CoverUploadID is an image file in the author's own drive; the
	// service resolves and counts the inode reference.
	CoverUploadID *uuid.UUID
	PublishAt     *time.Time
	ExpiresAt     *time.Time
}

// UpdateActivityInput is a partial edit of an activity; nil fields are left
// unchanged.
type UpdateActivityInput struct {
	ID            uuid.UUID
	ActorID       uuid.UUID // a replaced cover resolves from the actor's own drive
	TitleEN       *string
	TitleLocal    *string
	BodyEN        *string
	BodyLocal     *string
	Type          *ActivityType
	CoverUploadID *uuid.UUID
	PublishAt     *time.Time
	ExpiresAt     *time.Time
}

// AddActivityAttachmentInput attaches an uploaded file to an activity.
type AddActivityAttachmentInput struct {
	ActivityID  uuid.UUID
	ActorID     uuid.UUID
	UploadID    uuid.UUID
	DisplayName string
	FileType    string
	OrderIndex  int
}

// ── Service (use cases) ────────────────────────────────────────────────────

// ActivityService manages the institutional activity feed.
type ActivityService struct {
	repo       ActivityRepository
	publishers PublisherChecker
	langs      DefaultLanguageProvider
	files      FileStore
	log        *slog.Logger
}

// NewActivityService wires an activity service.
func NewActivityService(repo ActivityRepository, publishers PublisherChecker, langs DefaultLanguageProvider, files FileStore, log *slog.Logger) *ActivityService {
	return &ActivityService{repo: repo, publishers: publishers, langs: langs, files: files, log: log}
}

// Create publishes an activity after validating its publisher.
func (s *ActivityService) Create(ctx context.Context, in CreateActivityInput) (*Activity, error) {
	if !ValidPublisherType(in.PublisherType) || !ValidPublisherID(in.PublisherType, in.PublisherID) {
		return nil, ErrInvalidPublisher
	}
	if !ValidActivityType(in.Type) {
		return nil, ErrInvalidType
	}
	if in.PublisherID != nil {
		exists, err := s.publishers.PublisherExists(ctx, in.PublisherType, *in.PublisherID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, ErrInvalidPublisher
		}
	}

	a := BuildActivity(in)
	if in.CoverUploadID != nil {
		inodeID, err := s.resolveCover(ctx, in.AuthorID, *in.CoverUploadID)
		if err != nil {
			return nil, err
		}
		a.CoverImageID = inodeID
	}
	if err := s.repo.CreateActivity(ctx, a); err != nil {
		if a.CoverImageID != nil {
			s.unlink(ctx, *a.CoverImageID)
		}
		return nil, err
	}
	return a, nil
}

// resolveCover maps a drive file to its inode and takes a counted
// reference on it, refusing anything that is not an image.
func (s *ActivityService) resolveCover(ctx context.Context, ownerID, fileID uuid.UUID) (*uuid.UUID, error) {
	cover, err := s.files.ResolveUpload(ctx, ownerID, fileID)
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(cover.MimeType, "image/") {
		return nil, ErrInvalidFileType
	}
	if err := s.files.Link(ctx, cover.InodeID); err != nil {
		return nil, err
	}
	return &cover.InodeID, nil
}

// Get fetches one visible activity with its attachments.
func (s *ActivityService) Get(ctx context.Context, id uuid.UUID, revealHidden bool) (*ActivityWithAuthor, []ActivityAttachment, error) {
	a, err := s.repo.GetActivityByIDWithAuthor(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if a == nil {
		return nil, nil, ErrActivityNotFound
	}
	if !a.CanView(revealHidden, time.Now()) {
		return nil, nil, ErrActivityNotFound
	}
	attachments, err := s.repo.ListAttachmentsByActivityID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	return a, attachments, nil
}

// GetByID fetches the bare activity row.
func (s *ActivityService) GetByID(ctx context.Context, id uuid.UUID) (*Activity, error) {
	return s.repo.GetActivityByID(ctx, id)
}

// Translate returns the exact language variant of a visible activity.
func (s *ActivityService) Translate(ctx context.Context, id uuid.UUID, lang Lang, revealHidden bool) (title, body string, err error) {
	if !ValidLanguage(lang) {
		return "", "", ErrInvalidLanguage
	}
	a, err := s.repo.GetActivityByID(ctx, id)
	if err != nil {
		return "", "", err
	}
	if a == nil || !a.CanView(revealHidden, time.Now()) {
		return "", "", ErrActivityNotFound
	}
	title, body, ok := ResolveTranslation(a, lang)
	if !ok {
		return "", "", ErrTranslationMissing
	}
	return title, body, nil
}

// List pages through a publisher's activities.
func (s *ActivityService) List(ctx context.Context, pt PublisherType, publisherID *uuid.UUID, activityType ActivityType, revealHidden bool, params pagination.PageParams) ([]ActivityWithAuthor, bool, error) {
	return s.repo.ListActivitiesByPublisher(ctx, pt, publisherID, activityType, revealHidden, params)
}

// Update applies the edit to the actor's own activity under optimistic
// concurrency; a lost version race is retried against fresh state up to
// maxUpdateRetries times.
func (s *ActivityService) Update(ctx context.Context, in UpdateActivityInput) (*Activity, error) {
	for attempt := 0; attempt < maxUpdateRetries; attempt++ {
		a, err := s.repo.GetActivityByID(ctx, in.ID)
		if err != nil {
			return nil, err
		}
		if a == nil {
			return nil, ErrActivityNotFound
		}

		if in.TitleEN != nil {
			a.TitleEN = *in.TitleEN
		}
		if in.BodyEN != nil {
			a.BodyEN = *in.BodyEN
		}
		if in.TitleLocal != nil {
			a.TitleLocal = in.TitleLocal
		}
		if in.BodyLocal != nil {
			a.BodyLocal = in.BodyLocal
		}
		if in.Type != nil {
			if !ValidActivityType(*in.Type) {
				return nil, ErrInvalidType
			}
			a.Type = *in.Type
		}
		var oldCover, newCover *uuid.UUID
		if in.CoverUploadID != nil {
			oldCover = a.CoverImageID
			newCover, err = s.resolveCover(ctx, in.ActorID, *in.CoverUploadID)
			if err != nil {
				return nil, err
			}
			a.CoverImageID = newCover
		}
		if in.PublishAt != nil {
			a.PublishAt = in.PublishAt
		}
		if in.ExpiresAt != nil {
			a.ExpiresAt = in.ExpiresAt
		}
		now := time.Now()
		a.UpdatedAt = &now

		newVersion, err := s.repo.UpdateActivity(ctx, a, a.Version)
		if errors.Is(err, ErrConflict) {
			s.unlinkIf(ctx, newCover)
			continue
		}
		if err != nil {
			s.unlinkIf(ctx, newCover)
			return nil, err
		}
		a.Version = newVersion
		if newCover != nil {
			s.unlinkIf(ctx, oldCover)
		}
		return a, nil
	}
	return nil, ErrConflict
}

// Delete soft-deletes an activity. The authority was decided at the edge
// (§21).
func (s *ActivityService) Delete(ctx context.Context, id uuid.UUID) error {
	a, err := s.repo.GetActivityByID(ctx, id)
	if err != nil {
		return err
	}
	if a == nil {
		return ErrActivityNotFound
	}
	return s.repo.SoftDeleteActivity(ctx, id, time.Now())
}

// Pin sets or clears an activity's pin under optimistic concurrency. The
// pin authority was decided at the edge (§21). A lost version race is
// retried against fresh state.
func (s *ActivityService) Pin(ctx context.Context, id uuid.UUID, pin bool) error {
	for attempt := 0; attempt < maxUpdateRetries; attempt++ {
		a, err := s.repo.GetActivityByID(ctx, id)
		if err != nil {
			return err
		}
		if a == nil {
			return ErrActivityNotFound
		}
		a.IsPinned = pin
		now := time.Now()
		a.UpdatedAt = &now

		if _, err := s.repo.UpdateActivity(ctx, a, a.Version); errors.Is(err, ErrConflict) {
			continue
		} else if err != nil {
			return err
		}
		return nil
	}
	return ErrConflict
}

// AddAttachment references a file from the actor's own drive on their
// activity, counted before the row exists; type and size are judged from
// the stored facts.
func (s *ActivityService) AddAttachment(ctx context.Context, in AddActivityAttachmentInput) (*ActivityAttachment, error) {
	a, err := s.repo.GetActivityByID(ctx, in.ActivityID)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, ErrActivityNotFound
	}
	if !ValidFileType(in.FileType) {
		return nil, ErrInvalidFileType
	}
	file, err := s.files.ResolveUpload(ctx, in.ActorID, in.UploadID)
	if err != nil {
		return nil, err
	}
	if !ValidFileSize(in.FileType, file.SizeBytes) {
		return nil, ErrFileTooLarge
	}
	name := in.DisplayName
	if name == "" {
		name = file.Name
	}
	if err := s.files.Link(ctx, file.InodeID); err != nil {
		return nil, err
	}
	att := BuildActivityAttachment(in.ActivityID, file.InodeID, name, in.FileType, in.OrderIndex)
	if err := s.repo.CreateAttachment(ctx, att); err != nil {
		s.unlink(ctx, file.InodeID)
		return nil, err
	}
	return att, nil
}

// RemoveAttachment detaches a file from an activity and drops its
// reference count. The authority was decided at the edge (§21).
func (s *ActivityService) RemoveAttachment(ctx context.Context, attachmentID uuid.UUID) error {
	att, err := s.repo.GetAttachmentByID(ctx, attachmentID)
	if err != nil {
		return err
	}
	if att == nil {
		return ErrAttachmentNotFound
	}
	if err := s.repo.DeleteAttachment(ctx, attachmentID); err != nil {
		return err
	}
	s.unlink(ctx, att.InodeID)
	return nil
}

// PresignAttachment mints a download URL for one attachment of an activity
// the reader may see — whoever can read it can save its files.
func (s *ActivityService) PresignAttachment(ctx context.Context, activityID, attachmentID uuid.UUID, revealHidden bool) (string, error) {
	a, err := s.repo.GetActivityByID(ctx, activityID)
	if err != nil {
		return "", err
	}
	if a == nil || !a.CanView(revealHidden, time.Now()) {
		return "", ErrActivityNotFound
	}
	att, err := s.repo.GetAttachmentByID(ctx, attachmentID)
	if err != nil {
		return "", err
	}
	if att == nil || att.ActivityID != activityID {
		return "", ErrAttachmentNotFound
	}
	return s.files.Presign(ctx, att.InodeID, att.DisplayName)
}

// unlink drops one reference count; a failure over-counts (leaks a blob,
// never loses one), so it is logged, not fatal.
func (s *ActivityService) unlink(ctx context.Context, inodeID uuid.UUID) {
	if err := s.files.Unlink(ctx, inodeID); err != nil {
		s.log.WarnContext(ctx, "announcements: unlink failed; blob over-counted", "inode", inodeID, "error", err)
	}
}

func (s *ActivityService) unlinkIf(ctx context.Context, inodeID *uuid.UUID) {
	if inodeID != nil {
		s.unlink(ctx, *inodeID)
	}
}

// AttachmentByID fetches one attachment.
func (s *ActivityService) AttachmentByID(ctx context.Context, id uuid.UUID) (*ActivityAttachment, error) {
	return s.repo.GetAttachmentByID(ctx, id)
}

// AttachmentsFor returns the attachments of many activities keyed by
// activity ID.
func (s *ActivityService) AttachmentsFor(ctx context.Context, activityIDs []uuid.UUID) (map[uuid.UUID][]ActivityAttachment, error) {
	return s.repo.ListAttachmentsByActivityIDs(ctx, activityIDs)
}

// DefaultLanguage returns the institution's configured feed language,
// defaulting to English.
func (s *ActivityService) DefaultLanguage(ctx context.Context) (Lang, error) {
	v, err := s.langs.GetDefaultLanguage(ctx)
	if err != nil || v == "" {
		return LangEN, nil
	}
	return Lang(v), nil
}
