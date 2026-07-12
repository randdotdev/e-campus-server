package management

import "errors"

// ErrConflict is returned when an optimistic compare-and-swap loses the race
// (a concurrent writer changed the row). Callers retry; the HTTP layer surfaces
// it as 409 once retries are exhausted.
var ErrConflict = errors.New("resource was modified concurrently")

// Structure (college / department / program) errors.
var (
	ErrCollegeNotFound        = errors.New("college not found")
	ErrDepartmentNotFound     = errors.New("department not found")
	ErrProgramNotFound        = errors.New("program not found")
	ErrCodeExists             = errors.New("code already exists")
	ErrCollegeLimitReached    = errors.New("college limit reached")
	ErrDepartmentLimitReached = errors.New("department limit reached")
	ErrProgramLimitReached    = errors.New("program limit reached")
)

// Academic calendar (year / semester) errors.
var (
	ErrAcademicYearNotFound    = errors.New("academic year not found")
	ErrSemesterNotFound        = errors.New("semester not found")
	ErrDuplicateYear           = errors.New("academic year already exists")
	ErrDuplicateSemester       = errors.New("semester already exists")
	ErrInvalidStatus           = errors.New("invalid status")
	ErrInvalidStatusTransition = errors.New("invalid status transition")
	ErrSemesterNotFinalized    = errors.New("semester not finalized")
	ErrSemesterArchived        = errors.New("semester is archived")
	ErrOfferingsNotFinalized   = errors.New("not all offerings are finalized")
	ErrSemesterNotActive       = errors.New("semester must be upcoming or active")
)

// Curriculum / requirement errors.
var (
	ErrCurriculumNotFound  = errors.New("curriculum entry not found")
	ErrRequirementNotFound = errors.New("semester requirement not found")
	ErrCourseNotFound      = errors.New("course not found")
	ErrDuplicateCurriculum = errors.New("curriculum entry already exists")
)

// Settings errors.
var (
	ErrSettingsNotFound        = errors.New("settings not found")
	ErrMissingInstitutionName  = errors.New("institution name is required")
	ErrInvalidGradingDisplay   = errors.New("invalid grading display mode")
	ErrInvalidSemestersPerYear = errors.New("semesters per year must be 1, 2, or 3")
	ErrSettingsConflict        = errors.New("settings were modified concurrently")
)

// Application errors.
var (
	ErrApplicationNotFound        = errors.New("application not found")
	ErrDuplicateApplication       = errors.New("pending application already exists for this program and year")
	ErrProgramInactive            = errors.New("program is not accepting applications")
	ErrAgeTooYoung                = errors.New("applicant does not meet minimum age requirement")
	ErrAgeTooOld                  = errors.New("applicant exceeds maximum age requirement")
	ErrApplicationCannotUpdate    = errors.New("application cannot be updated in current status")
	ErrApplicationCannotWithdraw  = errors.New("application cannot be withdrawn in current status")
	ErrApplicationCannotReview    = errors.New("application cannot be reviewed in current status")
	ErrApplicationCannotReviewOwn = errors.New("cannot review own application")
	ErrApplicationAccessDenied    = errors.New("access denied")
)

// Enrollment errors.
var (
	ErrEnrollmentNotFound   = errors.New("enrollment not found")
	ErrAlreadyEnrolled      = errors.New("already enrolled")
	ErrNotEnrolled          = errors.New("not enrolled")
	ErrOfferingNotFound     = errors.New("offering not found")
	ErrCohortGroupNotFound  = errors.New("cohort group not found")
	ErrDuplicateCohortGroup = errors.New("cohort group already exists")
	ErrRequestNotFound      = errors.New("enrollment request not found")
	ErrDuplicateRequest     = errors.New("enrollment request already exists")
	ErrAlreadyReviewed      = errors.New("request already reviewed")
	ErrNoPrerequisite       = errors.New("course has no prerequisite")
	ErrPrerequisitePassed   = errors.New("prerequisite already passed")
	ErrCourseNotFailed      = errors.New("course not failed")
	ErrNotNaturalCohort     = errors.New("student not in natural cohort")
	ErrInvalidRequestType   = errors.New("invalid request type")
)

// Course / offering / section / teacher errors.
var (
	ErrTeacherNotFound    = errors.New("teacher not found")
	ErrDuplicateCode      = errors.New("course code already exists")
	ErrDuplicateOffering  = errors.New("offering already exists")
	ErrPrerequisiteNotMet = errors.New("prerequisite not met")
	ErrAlreadyTeacher     = errors.New("user is already a teacher")
)

// Student / leave errors.
var (
	ErrStudentNotFound      = errors.New("student not found")
	ErrLeaveNotFound        = errors.New("leave not found")
	ErrUserNotFound         = errors.New("user not found")
	ErrDuplicateStudent     = errors.New("student already exists")
	ErrInvalidLeaveType     = errors.New("invalid leave type")
	ErrAlreadyOnLeave       = errors.New("student is already on leave")
	ErrNotOnLeave           = errors.New("student is not on leave")
	ErrLeaveEnded           = errors.New("leave has already ended")
	ErrLeaveAlreadyApproved = errors.New("leave has already been approved")
)
