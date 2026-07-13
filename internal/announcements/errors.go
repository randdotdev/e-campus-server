package announcements

import "errors"

// Shared
var (
	ErrNotAuthorized      = errors.New("announcements: not authorized")
	ErrAttachmentNotFound = errors.New("announcements: attachment not found")
	ErrUploadNotFound     = errors.New("announcements: upload not found")
	ErrInvalidFileType    = errors.New("announcements: invalid file type")
	ErrFileTooLarge       = errors.New("announcements: file too large")
	// ErrConflict is a lost optimistic-concurrency race: the aggregate's
	// version changed between read and write (Shape 1).
	ErrConflict = errors.New("announcements: conflict")
)

// Activity
var (
	ErrActivityNotFound   = errors.New("announcements: activity not found")
	ErrInvalidPublisher   = errors.New("announcements: invalid publisher")
	ErrInvalidType        = errors.New("announcements: invalid activity type")
	ErrInvalidLanguage    = errors.New("announcements: invalid language")
	ErrTranslationMissing = errors.New("announcements: translation not available")
)

// Post
var (
	ErrPostNotFound     = errors.New("announcements: post not found")
	ErrInvalidScope     = errors.New("announcements: invalid scope")
	ErrPostScheduled    = errors.New("announcements: post not yet published")
	ErrPostExpired      = errors.New("announcements: post expired")
	ErrPostDeleted      = errors.New("announcements: post deleted")
	ErrAlreadyLiked     = errors.New("announcements: already liked")
	ErrNotLiked         = errors.New("announcements: not liked")
	ErrCannotPinComment = errors.New("announcements: cannot pin comment")
	ErrUserMuted        = errors.New("announcements: user is muted")
)
