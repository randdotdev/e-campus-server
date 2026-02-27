package attendance

import "errors"

var (
	ErrLessonNotFound            = errors.New("lesson not found")
	ErrAttendanceNotRequired     = errors.New("attendance not required for this lesson")
	ErrAttendanceNotFound        = errors.New("attendance record not found")
	ErrExcuseNotFound            = errors.New("excuse request not found")
	ErrExcuseAlreadyExists       = errors.New("excuse request already exists for this lesson")
	ErrExcuseAlreadyReviewed     = errors.New("excuse request already reviewed")
	ErrInvalidPercentage         = errors.New("percentage must be 0, 25, 50, 75, or 100")
	ErrInvalidExcuseStatus       = errors.New("status must be approved or rejected")
	ErrStudentNotEnrolled        = errors.New("student not enrolled in this offering")
	ErrCannotExcuseOwnAttendance = errors.New("cannot review own excuse request")
)
