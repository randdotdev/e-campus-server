package application

import "errors"

var (
	ErrApplicationNotFound  = errors.New("application not found")
	ErrDuplicateApplication = errors.New("pending application already exists for this program and year")
	ErrProgramNotFound      = errors.New("program not found")
	ErrProgramInactive      = errors.New("program is not accepting applications")
	ErrAgeTooYoung          = errors.New("applicant does not meet minimum age requirement")
	ErrAgeTooOld            = errors.New("applicant exceeds maximum age requirement")
	ErrCannotUpdate         = errors.New("application cannot be updated in current status")
	ErrCannotWithdraw       = errors.New("application cannot be withdrawn in current status")
	ErrCannotReviewOwn      = errors.New("cannot review own application")
	ErrInvalidStatus        = errors.New("invalid status transition")
	ErrAccessDenied         = errors.New("access denied")
)
