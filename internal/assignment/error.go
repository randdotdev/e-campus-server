package assignment

import "errors"

var (
	ErrAssignmentNotFound = errors.New("assignment not found")
	ErrSubmissionNotFound = errors.New("submission not found")
	ErrAttachmentNotFound = errors.New("attachment not found")
	ErrNotPublished       = errors.New("assignment not published")
	ErrSubmissionsClosed  = errors.New("submissions closed")
	ErrAlreadyGraded      = errors.New("submission already graded")
	ErrAlreadySubmitted   = errors.New("already submitted")
	ErrSubmissionExists   = errors.New("submission already exists")
	ErrNotDraft           = errors.New("can only delete draft submissions")
	ErrCannotModify       = errors.New("cannot modify submission")
	ErrNotEnrolled        = errors.New("student not enrolled")
	ErrInvalidScore       = errors.New("score must be between 0 and max score")
	ErrFileNotOwned       = errors.New("file not owned by student")
	ErrNoContent          = errors.New("submission must have content or files")
)
