package course

import "errors"

var (
	ErrCourseNotFound     = errors.New("course not found")
	ErrOfferingNotFound   = errors.New("offering not found")
	ErrSectionNotFound    = errors.New("section not found")
	ErrLessonNotFound     = errors.New("lesson not found")
	ErrTeacherNotFound    = errors.New("teacher not found")
	ErrDuplicateCode      = errors.New("course code already exists")
	ErrDuplicateOffering  = errors.New("offering already exists")
	ErrDuplicateSection   = errors.New("section order index already exists")
	ErrDuplicateLesson    = errors.New("lesson order index already exists")
	ErrPrerequisiteNotMet = errors.New("prerequisite not met")
	ErrSectionLocked      = errors.New("section is locked")
	ErrAlreadyTeacher     = errors.New("user is already a teacher")
	ErrSemesterNotFound   = errors.New("semester not found")
)
