package news

import (
	"time"

	"github.com/google/uuid"
)

func ValidatePublisherType(publisherType string) bool {
	switch publisherType {
	case PublisherUniversity, PublisherCollege, PublisherDepartment:
		return true
	}
	return false
}

func ValidatePublisherID(publisherType string, publisherID *uuid.UUID) bool {
	if publisherType == PublisherUniversity {
		return publisherID == nil
	}
	return publisherID != nil
}

func ValidateCategory(category string) bool {
	switch category {
	case CategoryAnnouncement, CategoryEvent, CategoryAchievement, CategoryAcademic, CategoryGeneral:
		return true
	}
	return false
}

func ValidateFileType(fileType string) bool {
	switch fileType {
	case FileTypeImage, FileTypeDocument, FileTypeVideo:
		return true
	}
	return false
}

func ValidateLanguage(lang string) bool {
	return lang == LangEN || lang == LangLocal
}

func IsScheduled(publishAt *time.Time, now time.Time) bool {
	return publishAt != nil && publishAt.After(now)
}

func IsExpired(expiresAt *time.Time, now time.Time) bool {
	return expiresAt != nil && expiresAt.Before(now)
}

func IsDeleted(deletedAt *time.Time) bool {
	return deletedAt != nil
}

func IsVisible(n *News, now time.Time) bool {
	if IsDeleted(n.DeletedAt) {
		return false
	}
	if IsScheduled(n.PublishAt, now) {
		return false
	}
	if IsExpired(n.ExpiresAt, now) {
		return false
	}
	return true
}

func CanView(n *News, isAdmin bool, now time.Time) bool {
	if IsDeleted(n.DeletedAt) {
		return isAdmin
	}
	if IsScheduled(n.PublishAt, now) {
		return isAdmin
	}
	if IsExpired(n.ExpiresAt, now) {
		return isAdmin
	}
	return true
}

func GetStatus(n *News, now time.Time) string {
	if IsScheduled(n.PublishAt, now) {
		return StatusScheduled
	}
	if IsExpired(n.ExpiresAt, now) {
		return StatusExpired
	}
	return StatusPublished
}

func CanEdit(n *News, userID uuid.UUID, isAdmin bool) bool {
	if n.AuthorID == userID {
		return true
	}
	return isAdmin
}

func CanDelete(n *News, userID uuid.UUID, isAdmin bool) bool {
	return CanEdit(n, userID, isAdmin)
}

func CanPin(isAdmin bool) bool {
	return isAdmin
}

func ResolveTitle(n *News, preferredLang, defaultLang string) string {
	if preferredLang == LangLocal {
		if n.TitleLocal != nil && *n.TitleLocal != "" {
			return *n.TitleLocal
		}
		if defaultLang == LangLocal {
			return n.TitleEN
		}
	}
	return n.TitleEN
}

func ResolveBody(n *News, preferredLang, defaultLang string) string {
	if preferredLang == LangLocal {
		if n.BodyLocal != nil && *n.BodyLocal != "" {
			return *n.BodyLocal
		}
		if defaultLang == LangLocal {
			return n.BodyEN
		}
	}
	return n.BodyEN
}

func GetTranslation(n *News, lang string) (title, body string, ok bool) {
	if lang == LangLocal {
		if n.TitleLocal == nil || n.BodyLocal == nil {
			return "", "", false
		}
		return *n.TitleLocal, *n.BodyLocal, true
	}
	return n.TitleEN, n.BodyEN, true
}

func BuildNews(authorID uuid.UUID, publisherType string, publisherID *uuid.UUID, category, titleEN string, titleLocal *string, bodyEN string, bodyLocal *string, coverImageID *uuid.UUID, publishAt, expiresAt *time.Time) *News {
	return &News{
		ID:            uuid.New(),
		PublisherType: publisherType,
		PublisherID:   publisherID,
		Category:      category,
		TitleEN:       titleEN,
		TitleLocal:    titleLocal,
		BodyEN:        bodyEN,
		BodyLocal:     bodyLocal,
		CoverImageID:  coverImageID,
		AuthorID:      authorID,
		PublishAt:     publishAt,
		ExpiresAt:     expiresAt,
		CreatedAt:     time.Now(),
	}
}

func BuildAttachment(newsID, storedFileID uuid.UUID, displayName, fileType string, orderIndex int) *NewsAttachment {
	return &NewsAttachment{
		ID:           uuid.New(),
		NewsID:       newsID,
		StoredFileID: storedFileID,
		DisplayName:  displayName,
		FileType:     fileType,
		OrderIndex:   orderIndex,
	}
}
