package activity

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/ranjdotdev/e-campus-server/internal/pagination"
)

type ActivityRepository interface {
	Create(ctx context.Context, a *Activity) error
	GetByID(ctx context.Context, id uuid.UUID) (*Activity, error)
	GetByIDWithAuthor(ctx context.Context, id uuid.UUID) (*ActivityWithAuthor, error)
	Update(ctx context.Context, a *Activity) error
	SoftDelete(ctx context.Context, id uuid.UUID, deletedAt time.Time) error

	ListByPublisher(ctx context.Context, publisherType string, publisherID *uuid.UUID, activityType string, isAdmin bool, params pagination.PageParams) ([]ActivityWithAuthor, bool, error)
}

type AttachmentRepository interface {
	Create(ctx context.Context, a *ActivityAttachment) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID) (*ActivityAttachment, error)
	ListByActivityID(ctx context.Context, activityID uuid.UUID) ([]ActivityAttachment, error)
	ListByActivityIDs(ctx context.Context, activityIDs []uuid.UUID) (map[uuid.UUID][]ActivityAttachment, error)
}

type PublisherChecker interface {
	PublisherExists(ctx context.Context, publisherType string, publisherID uuid.UUID) (bool, error)
}

type SettingsProvider interface {
	GetDefaultLanguage(ctx context.Context) (string, error)
}

type Service struct {
	activities  ActivityRepository
	attachments AttachmentRepository
	publishers  PublisherChecker
	settings    SettingsProvider
}

func NewService(
	activities ActivityRepository,
	attachments AttachmentRepository,
	publishers PublisherChecker,
	settings SettingsProvider,
) *Service {
	return &Service{
		activities:  activities,
		attachments: attachments,
		publishers:  publishers,
		settings:    settings,
	}
}

func (s *Service) CreateActivity(ctx context.Context, authorID uuid.UUID, publisherType string, publisherID *uuid.UUID, activityType, titleEN string, titleLocal *string, bodyEN string, bodyLocal *string, coverImageID *uuid.UUID, publishAt, expiresAt *time.Time) (*Activity, error) {
	if !ValidatePublisherType(publisherType) {
		return nil, ErrInvalidPublisher
	}
	if !ValidatePublisherID(publisherType, publisherID) {
		return nil, ErrInvalidPublisher
	}
	if !ValidateType(activityType) {
		return nil, ErrInvalidType
	}

	if publisherID != nil {
		exists, err := s.publishers.PublisherExists(ctx, publisherType, *publisherID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, ErrInvalidPublisher
		}
	}

	a := BuildActivity(authorID, publisherType, publisherID, activityType, titleEN, titleLocal, bodyEN, bodyLocal, coverImageID, publishAt, expiresAt)

	if err := s.activities.Create(ctx, a); err != nil {
		return nil, err
	}

	return a, nil
}

func (s *Service) GetActivity(ctx context.Context, id uuid.UUID, isAdmin bool) (*ActivityWithAuthor, []ActivityAttachment, error) {
	a, err := s.activities.GetByIDWithAuthor(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if a == nil {
		return nil, nil, ErrActivityNotFound
	}

	now := time.Now()
	if !CanView(&a.Activity, isAdmin, now) {
		return nil, nil, ErrActivityNotFound
	}

	attachments, err := s.attachments.ListByActivityID(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	return a, attachments, nil
}

func (s *Service) GetTranslation(ctx context.Context, id uuid.UUID, lang string, isAdmin bool) (string, string, error) {
	if !ValidateLanguage(lang) {
		return "", "", ErrInvalidLanguage
	}

	a, err := s.activities.GetByID(ctx, id)
	if err != nil {
		return "", "", err
	}
	if a == nil {
		return "", "", ErrActivityNotFound
	}

	now := time.Now()
	if !CanView(a, isAdmin, now) {
		return "", "", ErrActivityNotFound
	}

	title, body, ok := GetTranslation(a, lang)
	if !ok {
		return "", "", ErrTranslationMissing
	}

	return title, body, nil
}

func (s *Service) ListActivities(ctx context.Context, publisherType string, publisherID *uuid.UUID, activityType string, isAdmin bool, params pagination.PageParams) ([]ActivityWithAuthor, bool, error) {
	return s.activities.ListByPublisher(ctx, publisherType, publisherID, activityType, isAdmin, params)
}

func (s *Service) UpdateActivity(ctx context.Context, id, userID uuid.UUID, isAdmin bool, titleEN, bodyEN *string, titleLocal, bodyLocal *string, activityType *string, coverImageID *uuid.UUID, publishAt, expiresAt *time.Time) (*Activity, error) {
	a, err := s.activities.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, ErrActivityNotFound
	}

	if !CanEdit(a, userID, isAdmin) {
		return nil, ErrNotAuthorized
	}

	if titleEN != nil {
		a.TitleEN = *titleEN
	}
	if bodyEN != nil {
		a.BodyEN = *bodyEN
	}
	if titleLocal != nil {
		a.TitleLocal = titleLocal
	}
	if bodyLocal != nil {
		a.BodyLocal = bodyLocal
	}
	if activityType != nil {
		if !ValidateType(*activityType) {
			return nil, ErrInvalidType
		}
		a.Type = *activityType
	}
	if coverImageID != nil {
		a.CoverImageID = coverImageID
	}
	if publishAt != nil {
		a.PublishAt = publishAt
	}
	if expiresAt != nil {
		a.ExpiresAt = expiresAt
	}

	now := time.Now()
	a.UpdatedAt = &now

	if err := s.activities.Update(ctx, a); err != nil {
		return nil, err
	}

	return a, nil
}

func (s *Service) DeleteActivity(ctx context.Context, id, userID uuid.UUID, isAdmin bool) error {
	a, err := s.activities.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if a == nil {
		return ErrActivityNotFound
	}

	if !CanDelete(a, userID, isAdmin) {
		return ErrNotAuthorized
	}

	now := time.Now()
	return s.activities.SoftDelete(ctx, id, now)
}

func (s *Service) PinActivity(ctx context.Context, id uuid.UUID, isAdmin bool, pin bool) error {
	if !CanPin(isAdmin) {
		return ErrNotAuthorized
	}

	a, err := s.activities.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if a == nil {
		return ErrActivityNotFound
	}

	a.IsPinned = pin
	now := time.Now()
	a.UpdatedAt = &now

	return s.activities.Update(ctx, a)
}

func (s *Service) AddAttachment(ctx context.Context, activityID, userID uuid.UUID, isAdmin bool, storedFileID uuid.UUID, displayName, fileType string, sizeBytes int64, orderIndex int) (*ActivityAttachment, error) {
	a, err := s.activities.GetByID(ctx, activityID)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, ErrActivityNotFound
	}

	if !CanEdit(a, userID, isAdmin) {
		return nil, ErrNotAuthorized
	}

	if !ValidateFileType(fileType) {
		return nil, ErrInvalidFileType
	}

	if !ValidateFileSize(fileType, sizeBytes) {
		return nil, ErrFileTooLarge
	}

	attachment := BuildActivityAttachment(activityID, storedFileID, displayName, fileType, orderIndex)

	if err := s.attachments.Create(ctx, attachment); err != nil {
		return nil, err
	}

	return attachment, nil
}

func (s *Service) RemoveAttachment(ctx context.Context, attachmentID, userID uuid.UUID, isAdmin bool) error {
	attachment, err := s.attachments.GetByID(ctx, attachmentID)
	if err != nil {
		return err
	}
	if attachment == nil {
		return ErrAttachmentNotFound
	}

	a, err := s.activities.GetByID(ctx, attachment.ActivityID)
	if err != nil {
		return err
	}
	if a == nil {
		return ErrActivityNotFound
	}

	if !CanEdit(a, userID, isAdmin) {
		return ErrNotAuthorized
	}

	return s.attachments.Delete(ctx, attachmentID)
}

func (s *Service) GetAttachmentsForActivities(ctx context.Context, activityIDs []uuid.UUID) (map[uuid.UUID][]ActivityAttachment, error) {
	return s.attachments.ListByActivityIDs(ctx, activityIDs)
}

func (s *Service) GetActivityByID(ctx context.Context, id uuid.UUID) (*Activity, error) {
	return s.activities.GetByID(ctx, id)
}

func (s *Service) GetAttachmentByID(ctx context.Context, id uuid.UUID) (*ActivityAttachment, error) {
	return s.attachments.GetByID(ctx, id)
}

func (s *Service) GetDefaultLanguage(ctx context.Context) (string, error) {
	return s.settings.GetDefaultLanguage(ctx)
}
