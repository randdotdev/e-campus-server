package university

import "errors"

var (
	ErrCollegeNotFound        = errors.New("college not found")
	ErrDepartmentNotFound     = errors.New("department not found")
	ErrProgramNotFound        = errors.New("program not found")
	ErrCodeExists             = errors.New("code already exists")
	ErrCollegeLimitReached    = errors.New("college limit reached")
	ErrDepartmentLimitReached = errors.New("department limit reached")
	ErrProgramLimitReached    = errors.New("program limit reached")
)
