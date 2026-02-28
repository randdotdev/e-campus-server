package news

import "errors"

var (
	ErrNewsNotFound       = errors.New("news not found")
	ErrNotAuthorized      = errors.New("not authorized")
	ErrInvalidPublisher   = errors.New("invalid publisher")
	ErrInvalidCategory    = errors.New("invalid category")
	ErrAttachmentNotFound = errors.New("attachment not found")
	ErrInvalidFileType    = errors.New("invalid file type")
	ErrInvalidLanguage    = errors.New("invalid language")
	ErrTranslationMissing = errors.New("translation not available")
)
