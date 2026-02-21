package exam

import "errors"

var (
	ErrQuestionNotFound        = errors.New("question not found")
	ErrExamNotFound            = errors.New("exam not found")
	ErrAttemptNotFound         = errors.New("attempt not found")
	ErrOfferingNotFound        = errors.New("offering not found")
	ErrStudentNotFound         = errors.New("student not found")
	ErrNotEnrolled             = errors.New("student not enrolled in offering")
	ErrExamNotPublished        = errors.New("exam not published")
	ErrExamNotAvailable        = errors.New("exam not available")
	ErrExamClosed              = errors.New("exam is closed")
	ErrMaxAttemptsReached      = errors.New("max attempts reached")
	ErrAttemptAlreadyExists    = errors.New("attempt already exists")
	ErrAttemptNotStarted       = errors.New("attempt not started")
	ErrAttemptAlreadySubmitted = errors.New("attempt already submitted")
	ErrInvalidQuestionType     = errors.New("invalid question type")
	ErrInvalidExamType         = errors.New("invalid exam type")
	ErrInvalidExamMode         = errors.New("invalid exam mode")
	ErrInvalidDifficulty       = errors.New("invalid difficulty")
	ErrInvalidVisibility       = errors.New("invalid visibility")
	ErrCannotModifyPublished   = errors.New("cannot modify published exam")
	ErrNoQuestionsInExam       = errors.New("exam has no questions")
)
