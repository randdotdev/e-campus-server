package management

import "errors"

// ErrConflict is returned when an optimistic compare-and-swap loses the race
// (a concurrent writer changed the row). Callers retry; the HTTP layer surfaces
// it as 409 once retries are exhausted.
var ErrConflict = errors.New("management: resource was modified concurrently")

// Structure (college / department / program) errors.
var (
	ErrCollegeNotFound        = errors.New("management: college not found")
	ErrDepartmentNotFound     = errors.New("management: department not found")
	ErrProgramNotFound        = errors.New("management: program not found")
	ErrCodeExists             = errors.New("management: code already exists")
	ErrCollegeLimitReached    = errors.New("management: college limit reached")
	ErrDepartmentLimitReached = errors.New("management: department limit reached")
	ErrProgramLimitReached    = errors.New("management: program limit reached")
)

// Academic calendar (year / semester) errors.
var (
	ErrAcademicYearNotFound    = errors.New("management: academic year not found")
	ErrSemesterNotFound        = errors.New("management: semester not found")
	ErrDuplicateYear           = errors.New("management: academic year already exists")
	ErrDuplicateSemester       = errors.New("management: semester already exists")
	ErrInvalidStatus           = errors.New("management: invalid status")
	ErrInvalidStatusTransition = errors.New("management: invalid status transition")
	ErrSemesterNotFinalized    = errors.New("management: semester not finalized")
	ErrSemesterArchived        = errors.New("management: semester is archived")
	ErrOfferingsNotFinalized   = errors.New("management: not all offerings are finalized")
	ErrSemesterNotActive       = errors.New("management: semester must be upcoming or active")
)

// Curriculum / requirement errors.
var (
	ErrCurriculumNotFound  = errors.New("management: curriculum entry not found")
	ErrRequirementNotFound = errors.New("management: semester requirement not found")
	ErrCourseNotFound      = errors.New("management: course not found")
	ErrDuplicateCurriculum = errors.New("management: curriculum entry already exists")
)

// Settings errors.
var (
	ErrSettingsNotFound        = errors.New("management: settings not found")
	ErrMissingInstitutionName  = errors.New("management: institution name is required")
	ErrInvalidGradingDisplay   = errors.New("management: invalid grading display mode")
	ErrInvalidSemestersPerYear = errors.New("management: semesters per year must be 1, 2, or 3")
	ErrSettingsConflict        = errors.New("management: settings were modified concurrently")
)

// Application errors.
var (
	ErrApplicationNotFound        = errors.New("management: application not found")
	ErrDuplicateApplication       = errors.New("management: pending application already exists for this program and year")
	ErrProgramInactive            = errors.New("management: program is not accepting applications")
	ErrAgeTooYoung                = errors.New("management: applicant does not meet minimum age requirement")
	ErrAgeTooOld                  = errors.New("management: applicant exceeds maximum age requirement")
	ErrApplicationCannotUpdate    = errors.New("management: application cannot be updated in current status")
	ErrApplicationCannotWithdraw  = errors.New("management: application cannot be withdrawn in current status")
	ErrApplicationCannotReview    = errors.New("management: application cannot be reviewed in current status")
	ErrApplicationCannotReviewOwn = errors.New("management: cannot review own application")
	ErrApplicationAccessDenied    = errors.New("management: access denied")
)

// Enrollment errors.
var (
	ErrEnrollmentNotFound   = errors.New("management: enrollment not found")
	ErrAlreadyEnrolled      = errors.New("management: already enrolled")
	ErrNotEnrolled          = errors.New("management: not enrolled")
	ErrOfferingNotFound     = errors.New("management: offering not found")
	ErrCohortGroupNotFound  = errors.New("management: cohort group not found")
	ErrDuplicateCohortGroup = errors.New("management: cohort group already exists")
	ErrRequestNotFound      = errors.New("management: enrollment request not found")
	ErrDuplicateRequest     = errors.New("management: enrollment request already exists")
	ErrAlreadyReviewed      = errors.New("management: request already reviewed")
	ErrNoPrerequisite       = errors.New("management: course has no prerequisite")
	ErrPrerequisitePassed   = errors.New("management: prerequisite already passed")
	ErrCourseNotFailed      = errors.New("management: course not failed")
	ErrNotNaturalCohort     = errors.New("management: student not in natural cohort")
	ErrInvalidRequestType   = errors.New("management: invalid request type")
)

// Course / offering / section / teacher errors.
var (
	ErrTeacherNotFound    = errors.New("management: teacher not found")
	ErrDuplicateCode      = errors.New("management: course code already exists")
	ErrDuplicateOffering  = errors.New("management: offering already exists")
	ErrPrerequisiteNotMet = errors.New("management: prerequisite not met")
	ErrAlreadyTeacher     = errors.New("management: user is already a teacher")
)

// Student / leave errors.
var (
	ErrStudentNotFound      = errors.New("management: student not found")
	ErrLeaveNotFound        = errors.New("management: leave not found")
	ErrUserNotFound         = errors.New("management: user not found")
	ErrDuplicateStudent     = errors.New("management: student already exists")
	ErrInvalidLeaveType     = errors.New("management: invalid leave type")
	ErrAlreadyOnLeave       = errors.New("management: student is already on leave")
	ErrNotOnLeave           = errors.New("management: student is not on leave")
	ErrLeaveEnded           = errors.New("management: leave has already ended")
	ErrLeaveAlreadyApproved = errors.New("management: leave has already been approved")
)
