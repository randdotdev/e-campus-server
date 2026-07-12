package announcements

import "errors"

// Shared
var (
	ErrNotAuthorized      = errors.New("not authorized")
	ErrAttachmentNotFound = errors.New("attachment not found")
	ErrUploadNotFound     = errors.New("announcements: upload not found")
	ErrInvalidFileType    = errors.New("invalid file type")
	ErrFileTooLarge       = errors.New("file too large")
	// ErrConflict is a lost optimistic-concurrency race: the aggregate's
	// version changed between read and write (Shape 1).
	ErrConflict = errors.New("conflict")
)

// Activity
var (
	ErrActivityNotFound   = errors.New("activity not found")
	ErrInvalidPublisher   = errors.New("invalid publisher")
	ErrInvalidType        = errors.New("invalid activity type")
	ErrInvalidLanguage    = errors.New("invalid language")
	ErrTranslationMissing = errors.New("translation not available")
)

// Post
var (
	ErrPostNotFound     = errors.New("post not found")
	ErrInvalidScope     = errors.New("invalid scope")
	ErrPostScheduled    = errors.New("post not yet published")
	ErrPostExpired      = errors.New("post expired")
	ErrPostDeleted      = errors.New("post deleted")
	ErrAlreadyLiked     = errors.New("already liked")
	ErrNotLiked         = errors.New("not liked")
	ErrCannotPinComment = errors.New("cannot pin comment")
	ErrUserMuted        = errors.New("user is muted")
)
