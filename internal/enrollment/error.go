package enrollment

import "errors"

var (
	ErrRequestNotFound     = errors.New("enrollment request not found")
	ErrDuplicateRequest    = errors.New("enrollment request already exists")
	ErrAlreadyReviewed     = errors.New("request already reviewed")
	ErrCourseNotFound      = errors.New("course not found")
	ErrSemesterNotFound    = errors.New("semester not found")
	ErrNoPrerequisite      = errors.New("course has no prerequisite")
	ErrPrerequisitePassed  = errors.New("prerequisite already passed")
	ErrCourseNotFailed     = errors.New("course not failed")
	ErrNotNaturalCohort    = errors.New("student not in natural cohort")
	ErrInvalidRequestType  = errors.New("invalid request type")
)
