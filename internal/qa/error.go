package qa

import "errors"

var (
	ErrQuestionNotFound = errors.New("question not found")
	ErrAnswerNotFound   = errors.New("answer not found")
	ErrOfferingNotFound = errors.New("offering not found")
	ErrNotAuthor        = errors.New("not question author")
	ErrNotAuthorized    = errors.New("not authorized")
	ErrQuestionAnswered = errors.New("question already answered")
	ErrQuestionRejected = errors.New("question rejected")
	ErrQuestionPending  = errors.New("question pending")
	ErrNotPending       = errors.New("question not pending")
	ErrUserMuted        = errors.New("user is muted")
	ErrEmptyTitle       = errors.New("title required")
	ErrEmptyBody        = errors.New("body required")
	ErrTitleTooLong     = errors.New("title too long")
	ErrEmptyReason      = errors.New("rejection reason required")
)
