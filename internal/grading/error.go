package grading

import "errors"

var (
	ErrOfferingNotFound    = errors.New("offering not found")
	ErrRulesNotFound       = errors.New("grading rules not found")
	ErrInvalidRuleType     = errors.New("invalid rule type")
	ErrWeightsMustSum100   = errors.New("weights must sum to 100")
	ErrExamNotFound        = errors.New("exam not found")
	ErrSemesterNotGrading  = errors.New("semester must be in grading status")
	ErrSemesterArchived    = errors.New("cannot modify archived semester")
	ErrAlreadyFinalized    = errors.New("grades already finalized")
	ErrNotFinalized        = errors.New("grades not finalized")
	ErrNoEnrollments       = errors.New("no enrolled students")
	ErrStudentNotEnrolled  = errors.New("student not enrolled")
	ErrInvalidGrade        = errors.New("grade must be between 0 and 100")
	ErrUngradedExams       = errors.New("some exams have ungraded submissions")
	ErrUngradedAssignments = errors.New("some assignments have ungraded submissions")
)
