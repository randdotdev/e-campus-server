package activity

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

func ValidateType(activityType string) bool {
	switch activityType {
	case TypeNews, TypeAnnouncement, TypeWebinar, TypeWorkshop, TypeConference, TypeSymposium, TypeTrainingCourse:
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

func ValidateFileSize(fileType string, sizeBytes int64) bool {
	switch fileType {
	case FileTypeImage:
		return sizeBytes <= MaxImageSize
	case FileTypeVideo:
		return sizeBytes <= MaxVideoSize
	case FileTypeDocument:
		return sizeBytes <= MaxDocumentSize
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

func IsVisible(a *Activity, now time.Time) bool {
	if IsDeleted(a.DeletedAt) {
		return false
	}
	if IsScheduled(a.PublishAt, now) {
		return false
	}
	if IsExpired(a.ExpiresAt, now) {
		return false
	}
	return true
}

func CanView(a *Activity, isAdmin bool, now time.Time) bool {
	if IsDeleted(a.DeletedAt) {
		return isAdmin
	}
	if IsScheduled(a.PublishAt, now) {
		return isAdmin
	}
	if IsExpired(a.ExpiresAt, now) {
		return isAdmin
	}
	return true
}

func GetStatus(a *Activity, now time.Time) string {
	if IsScheduled(a.PublishAt, now) {
		return StatusScheduled
	}
	if IsExpired(a.ExpiresAt, now) {
		return StatusExpired
	}
	return StatusPublished
}

func CanEdit(a *Activity, userID uuid.UUID, isAdmin bool) bool {
	if a.AuthorID == userID {
		return true
	}
	return isAdmin
}

func CanDelete(a *Activity, userID uuid.UUID, isAdmin bool) bool {
	return CanEdit(a, userID, isAdmin)
}

func CanPin(isAdmin bool) bool {
	return isAdmin
}

func ResolveTitle(a *Activity, preferredLang, defaultLang string) string {
	if preferredLang == LangLocal {
		if a.TitleLocal != nil && *a.TitleLocal != "" {
			return *a.TitleLocal
		}
		if defaultLang == LangLocal {
			return a.TitleEN
		}
	}
	return a.TitleEN
}

func ResolveBody(a *Activity, preferredLang, defaultLang string) string {
	if preferredLang == LangLocal {
		if a.BodyLocal != nil && *a.BodyLocal != "" {
			return *a.BodyLocal
		}
		if defaultLang == LangLocal {
			return a.BodyEN
		}
	}
	return a.BodyEN
}

func GetTranslation(a *Activity, lang string) (title, body string, ok bool) {
	if lang == LangLocal {
		if a.TitleLocal == nil || a.BodyLocal == nil {
			return "", "", false
		}
		return *a.TitleLocal, *a.BodyLocal, true
	}
	return a.TitleEN, a.BodyEN, true
}

func BuildActivity(authorID uuid.UUID, publisherType string, publisherID *uuid.UUID, activityType, titleEN string, titleLocal *string, bodyEN string, bodyLocal *string, coverImageID *uuid.UUID, publishAt, expiresAt *time.Time) *Activity {
	return &Activity{
		ID:            uuid.New(),
		PublisherType: publisherType,
		PublisherID:   publisherID,
		Type:          activityType,
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

func BuildActivityAttachment(activityID, storedFileID uuid.UUID, displayName, fileType string, orderIndex int) *ActivityAttachment {
	return &ActivityAttachment{
		ID:           uuid.New(),
		ActivityID:   activityID,
		StoredFileID: storedFileID,
		DisplayName:  displayName,
		FileType:     fileType,
		OrderIndex:   orderIndex,
	}
}
