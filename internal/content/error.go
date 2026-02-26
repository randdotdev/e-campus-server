package content

import "errors"

var (
	ErrSectionNotFound       = errors.New("section not found")
	ErrLessonNotFound        = errors.New("lesson not found")
	ErrAttachmentNotFound    = errors.New("attachment not found")
	ErrScheduleNotFound      = errors.New("schedule not found")
	ErrDuplicateDisplayName  = errors.New("attachment with this display name already exists")
	ErrDuplicateSchedule     = errors.New("schedule for this group already exists")
	ErrInvalidMode           = errors.New("invalid lesson mode")
	ErrInvalidType           = errors.New("invalid lesson type")
	ErrSectionNotEmpty       = errors.New("section contains lessons")
	ErrStoredFileNotFound    = errors.New("stored file not found")
	ErrOfferingNotFound      = errors.New("offering not found")
	ErrGroupNotFound         = errors.New("group not found")
	ErrNoAccess              = errors.New("no access to this content")
)
