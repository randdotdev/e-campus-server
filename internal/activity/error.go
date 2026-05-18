package activity

import "errors"

var (
	ErrActivityNotFound   = errors.New("activity not found")
	ErrNotAuthorized      = errors.New("not authorized")
	ErrInvalidPublisher   = errors.New("invalid publisher")
	ErrInvalidType        = errors.New("invalid activity type")
	ErrAttachmentNotFound = errors.New("attachment not found")
	ErrInvalidFileType    = errors.New("invalid file type")
	ErrFileTooLarge       = errors.New("file too large")
	ErrInvalidLanguage    = errors.New("invalid language")
	ErrTranslationMissing = errors.New("translation not available")
)
