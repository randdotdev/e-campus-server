package news

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/ranjdotdev/e-campus-server/internal/pagination"
)

type NewsRepository interface {
	Create(ctx context.Context, n *News) error
	GetByID(ctx context.Context, id uuid.UUID) (*News, error)
	GetByIDWithAuthor(ctx context.Context, id uuid.UUID) (*NewsWithAuthor, error)
	Update(ctx context.Context, n *News) error
	SoftDelete(ctx context.Context, id uuid.UUID, deletedAt time.Time) error

	ListByPublisher(ctx context.Context, publisherType string, publisherID *uuid.UUID, category string, isAdmin bool, params pagination.PageParams) ([]NewsWithAuthor, bool, error)
}

type AttachmentRepository interface {
	Create(ctx context.Context, a *NewsAttachment) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID) (*NewsAttachment, error)
	ListByNewsID(ctx context.Context, newsID uuid.UUID) ([]NewsAttachment, error)
	ListByNewsIDs(ctx context.Context, newsIDs []uuid.UUID) (map[uuid.UUID][]NewsAttachment, error)
}

type PublisherChecker interface {
	PublisherExists(ctx context.Context, publisherType string, publisherID uuid.UUID) (bool, error)
}

type SettingsProvider interface {
	GetDefaultLanguage(ctx context.Context) (string, error)
}

type Service struct {
	news        NewsRepository
	attachments AttachmentRepository
	publishers  PublisherChecker
	settings    SettingsProvider
}

func NewService(
	news NewsRepository,
	attachments AttachmentRepository,
	publishers PublisherChecker,
	settings SettingsProvider,
) *Service {
	return &Service{
		news:        news,
		attachments: attachments,
		publishers:  publishers,
		settings:    settings,
	}
}

func (s *Service) CreateNews(ctx context.Context, authorID uuid.UUID, publisherType string, publisherID *uuid.UUID, category, titleEN string, titleLocal *string, bodyEN string, bodyLocal *string, coverImageID *uuid.UUID, publishAt, expiresAt *time.Time) (*News, error) {
	if !ValidatePublisherType(publisherType) {
		return nil, ErrInvalidPublisher
	}
	if !ValidatePublisherID(publisherType, publisherID) {
		return nil, ErrInvalidPublisher
	}
	if !ValidateCategory(category) {
		return nil, ErrInvalidCategory
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

	news := BuildNews(authorID, publisherType, publisherID, category, titleEN, titleLocal, bodyEN, bodyLocal, coverImageID, publishAt, expiresAt)

	if err := s.news.Create(ctx, news); err != nil {
		return nil, err
	}

	return news, nil
}

func (s *Service) GetNews(ctx context.Context, id uuid.UUID, isAdmin bool) (*NewsWithAuthor, []NewsAttachment, error) {
	news, err := s.news.GetByIDWithAuthor(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if news == nil {
		return nil, nil, ErrNewsNotFound
	}

	now := time.Now()
	if !CanView(&news.News, isAdmin, now) {
		return nil, nil, ErrNewsNotFound
	}

	attachments, err := s.attachments.ListByNewsID(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	return news, attachments, nil
}

func (s *Service) GetTranslation(ctx context.Context, id uuid.UUID, lang string, isAdmin bool) (string, string, error) {
	if !ValidateLanguage(lang) {
		return "", "", ErrInvalidLanguage
	}

	news, err := s.news.GetByID(ctx, id)
	if err != nil {
		return "", "", err
	}
	if news == nil {
		return "", "", ErrNewsNotFound
	}

	now := time.Now()
	if !CanView(news, isAdmin, now) {
		return "", "", ErrNewsNotFound
	}

	title, body, ok := GetTranslation(news, lang)
	if !ok {
		return "", "", ErrTranslationMissing
	}

	return title, body, nil
}

func (s *Service) ListNews(ctx context.Context, publisherType string, publisherID *uuid.UUID, category string, isAdmin bool, params pagination.PageParams) ([]NewsWithAuthor, bool, error) {
	return s.news.ListByPublisher(ctx, publisherType, publisherID, category, isAdmin, params)
}

func (s *Service) UpdateNews(ctx context.Context, id, userID uuid.UUID, isAdmin bool, titleEN, bodyEN *string, titleLocal, bodyLocal *string, category *string, coverImageID *uuid.UUID, publishAt, expiresAt *time.Time) (*News, error) {
	news, err := s.news.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if news == nil {
		return nil, ErrNewsNotFound
	}

	if !CanEdit(news, userID, isAdmin) {
		return nil, ErrNotAuthorized
	}

	if titleEN != nil {
		news.TitleEN = *titleEN
	}
	if bodyEN != nil {
		news.BodyEN = *bodyEN
	}
	if titleLocal != nil {
		news.TitleLocal = titleLocal
	}
	if bodyLocal != nil {
		news.BodyLocal = bodyLocal
	}
	if category != nil {
		if !ValidateCategory(*category) {
			return nil, ErrInvalidCategory
		}
		news.Category = *category
	}
	if coverImageID != nil {
		news.CoverImageID = coverImageID
	}
	if publishAt != nil {
		news.PublishAt = publishAt
	}
	if expiresAt != nil {
		news.ExpiresAt = expiresAt
	}

	now := time.Now()
	news.UpdatedAt = &now

	if err := s.news.Update(ctx, news); err != nil {
		return nil, err
	}

	return news, nil
}

func (s *Service) DeleteNews(ctx context.Context, id, userID uuid.UUID, isAdmin bool) error {
	news, err := s.news.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if news == nil {
		return ErrNewsNotFound
	}

	if !CanDelete(news, userID, isAdmin) {
		return ErrNotAuthorized
	}

	now := time.Now()
	return s.news.SoftDelete(ctx, id, now)
}

func (s *Service) PinNews(ctx context.Context, id uuid.UUID, isAdmin bool, pin bool) error {
	if !CanPin(isAdmin) {
		return ErrNotAuthorized
	}

	news, err := s.news.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if news == nil {
		return ErrNewsNotFound
	}

	news.IsPinned = pin
	now := time.Now()
	news.UpdatedAt = &now

	return s.news.Update(ctx, news)
}

func (s *Service) AddAttachment(ctx context.Context, newsID, userID uuid.UUID, isAdmin bool, storedFileID uuid.UUID, displayName, fileType string, orderIndex int) (*NewsAttachment, error) {
	news, err := s.news.GetByID(ctx, newsID)
	if err != nil {
		return nil, err
	}
	if news == nil {
		return nil, ErrNewsNotFound
	}

	if !CanEdit(news, userID, isAdmin) {
		return nil, ErrNotAuthorized
	}

	if !ValidateFileType(fileType) {
		return nil, ErrInvalidFileType
	}

	attachment := BuildAttachment(newsID, storedFileID, displayName, fileType, orderIndex)

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

	news, err := s.news.GetByID(ctx, attachment.NewsID)
	if err != nil {
		return err
	}
	if news == nil {
		return ErrNewsNotFound
	}

	if !CanEdit(news, userID, isAdmin) {
		return ErrNotAuthorized
	}

	return s.attachments.Delete(ctx, attachmentID)
}

func (s *Service) GetAttachmentsForNews(ctx context.Context, newsIDs []uuid.UUID) (map[uuid.UUID][]NewsAttachment, error) {
	return s.attachments.ListByNewsIDs(ctx, newsIDs)
}

func (s *Service) GetNewsByID(ctx context.Context, id uuid.UUID) (*News, error) {
	return s.news.GetByID(ctx, id)
}

func (s *Service) GetAttachmentByID(ctx context.Context, id uuid.UUID) (*NewsAttachment, error) {
	return s.attachments.GetByID(ctx, id)
}

func (s *Service) GetDefaultLanguage(ctx context.Context) (string, error) {
	return s.settings.GetDefaultLanguage(ctx)
}
