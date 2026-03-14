package student

import "errors"

var (
	ErrStudentNotFound      = errors.New("student not found")
	ErrLeaveNotFound        = errors.New("leave not found")
	ErrUserNotFound         = errors.New("user not found")
	ErrProgramNotFound      = errors.New("program not found")
	ErrDuplicateStudent     = errors.New("student already exists for this user")
	ErrInvalidStatus        = errors.New("invalid status")
	ErrInvalidLeaveType     = errors.New("invalid leave type")
	ErrAlreadyOnLeave       = errors.New("student is already on leave")
	ErrNotOnLeave           = errors.New("student is not on leave")
	ErrLeaveEnded           = errors.New("leave has already ended")
	ErrLeaveAlreadyApproved = errors.New("leave has already been approved")
)
