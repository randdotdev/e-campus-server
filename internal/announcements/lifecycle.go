// Package announcements provides the platform feed. The feed is composed of two
// feed types: posts (member posts and classroom posts) and activities
// (institutional news, announcements, and events). It defines entities, ports,
// rules, and application services, and depends on no infrastructure.
package announcements

import "time"

// Status is the publish lifecycle state shared by activities and posts.
type Status string

const (
	StatusScheduled Status = "scheduled"
	StatusPublished Status = "published"
	StatusExpired   Status = "expired"
)

// Attachment file kinds and limits are shared mechanics. They stay untyped
// strings until the file-management package owns them.
const (
	FileTypeImage    = "image"
	FileTypeDocument = "document"
	FileTypeVoice    = "voice"
	FileTypeVideo    = "video"
)

const (
	MaxImageSize    = 10 * 1024 * 1024 // 10MB
	MaxVideoSize    = 50 * 1024 * 1024 // 50MB
	MaxVoiceSize    = 10 * 1024 * 1024 // 10MB
	MaxDocumentSize = 20 * 1024 * 1024 // 20MB
)

func ValidFileType(fileType string) bool {
	switch fileType {
	case FileTypeImage, FileTypeDocument, FileTypeVoice, FileTypeVideo:
		return true
	}
	return false
}

func ValidFileSize(fileType string, sizeBytes int64) bool {
	switch fileType {
	case FileTypeImage:
		return sizeBytes <= MaxImageSize
	case FileTypeVideo:
		return sizeBytes <= MaxVideoSize
	case FileTypeVoice:
		return sizeBytes <= MaxVoiceSize
	case FileTypeDocument:
		return sizeBytes <= MaxDocumentSize
	}
	return false
}

// Shared publish-window predicates. They operate on raw timestamps so both
// Activity and Post reuse them without a shared base type.

func IsScheduled(publishAt *time.Time, now time.Time) bool {
	return publishAt != nil && publishAt.After(now)
}

func IsExpired(expiresAt *time.Time, now time.Time) bool {
	return expiresAt != nil && expiresAt.Before(now)
}

func IsDeleted(deletedAt *time.Time) bool {
	return deletedAt != nil
}

func visible(deletedAt, publishAt, expiresAt *time.Time, now time.Time) bool {
	return !IsDeleted(deletedAt) && !IsScheduled(publishAt, now) && !IsExpired(expiresAt, now)
}

func canView(deletedAt, publishAt, expiresAt *time.Time, revealHidden bool, now time.Time) bool {
	if visible(deletedAt, publishAt, expiresAt, now) {
		return true
	}
	return revealHidden
}

func statusOf(publishAt, expiresAt *time.Time, now time.Time) Status {
	if IsScheduled(publishAt, now) {
		return StatusScheduled
	}
	if IsExpired(expiresAt, now) {
		return StatusExpired
	}
	return StatusPublished
}
