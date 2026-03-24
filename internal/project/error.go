package project

import "errors"

var (
	ErrProjectNotFound      = errors.New("project not found")
	ErrNotPublished         = errors.New("project not published")
	ErrNotEnrolled          = errors.New("not enrolled in course")
	ErrNotTeacher           = errors.New("teacher access required")
	ErrRegistrationClosed   = errors.New("registration deadline passed")
	ErrAlreadyRegistered    = errors.New("team already registered")
	ErrNotRegistered        = errors.New("team not registered")
	ErrNotTeamLeader        = errors.New("only team leader can register")
	ErrTeamTooSmall         = errors.New("team has fewer than minimum members")
	ErrTeamTooLarge         = errors.New("team has more than maximum members")
	ErrGroupNotFound        = errors.New("course group not found")
	ErrGroupFinalized       = errors.New("course group is finalized")
	ErrNotGroupMember       = errors.New("not a member of this group")
	ErrNotGroupLeader       = errors.New("only group leader can submit")
	ErrSubmissionNotFound   = errors.New("submission not found")
	ErrSubmissionsClosed    = errors.New("submission deadline passed")
	ErrAlreadySubmitted     = errors.New("already submitted")
	ErrNoContent            = errors.New("submission must have content or files")
	ErrInvalidScore         = errors.New("score out of range")
	ErrAttachmentNotFound   = errors.New("attachment not found")
	ErrFileNotOwned         = errors.New("file not owned by user")
	ErrMembersNotEnrolled   = errors.New("some team members not enrolled")
)
