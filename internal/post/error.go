package post

import "errors"

var (
	ErrCannotCommentOnComment = errors.New("cannot comment on comment")
	ErrPostNotFound           = errors.New("post not found")
	ErrNotAuthor              = errors.New("not post author")
	ErrNotAuthorized          = errors.New("not authorized")
	ErrInvalidScope           = errors.New("invalid scope")
	ErrInvalidParent          = errors.New("invalid parent post")
	ErrPostScheduled          = errors.New("post not yet published")
	ErrPostExpired            = errors.New("post expired")
	ErrPostDeleted            = errors.New("post deleted")
	ErrAlreadyLiked           = errors.New("already liked")
	ErrNotLiked               = errors.New("not liked")
	ErrAttachmentNotFound     = errors.New("attachment not found")
	ErrInvalidFileType        = errors.New("invalid file type")
	ErrFileTooLarge           = errors.New("file too large")
	ErrUserNotFound           = errors.New("user not found")
	ErrCannotPinComment       = errors.New("cannot pin comment")
	ErrUserMuted              = errors.New("user is muted")
)
