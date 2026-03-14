package academic

import "errors"

var (
	ErrAcademicYearNotFound   = errors.New("academic year not found")
	ErrSemesterNotFound       = errors.New("semester not found")
	ErrCurriculumNotFound     = errors.New("curriculum entry not found")
	ErrRequirementNotFound    = errors.New("semester requirement not found")
	ErrProgramNotFound        = errors.New("program not found")
	ErrCourseNotFound         = errors.New("course not found")
	ErrDuplicateYear          = errors.New("academic year already exists")
	ErrDuplicateSemester      = errors.New("semester already exists")
	ErrDuplicateCurriculum    = errors.New("curriculum entry already exists")
	ErrInvalidStatus          = errors.New("invalid status")
	ErrInvalidStatusTransition = errors.New("invalid status transition")
	ErrSemesterNotFinalized   = errors.New("semester not finalized")
	ErrSemesterArchived       = errors.New("semester is archived")
	ErrOfferingsNotFinalized  = errors.New("not all offerings are finalized")
)
