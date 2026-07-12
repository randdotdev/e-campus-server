// Package http is the management context's HTTP surface: request binding,
// response DTOs, and the single error→status translation point.
package http

import (
	"errors"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/randdotdev/e-campus-server/internal/authz"
	authzhttp "github.com/randdotdev/e-campus-server/internal/authz/http"
	"github.com/randdotdev/e-campus-server/internal/management"
	"github.com/randdotdev/e-campus-server/internal/shared/response"
)

// Handler is the management context's HTTP surface. It fans out to the
// per-noun services; each noun's handlers and DTOs live in its own file.
type Handler struct {
	colleges     *management.CollegeService
	departments  *management.DepartmentService
	programs     *management.ProgramService
	years        *management.AcademicYearService
	semesters    *management.SemesterService
	curriculum   *management.CurriculumService
	settings     *management.SettingsService
	applications *management.ApplicationService
	enrollment   *management.EnrollmentService
	cohortGroups *management.CohortGroupService
	requests     *management.RequestService
	courses      *management.CourseService
	offerings    *management.OfferingService
	teachers     *management.TeacherService
	students     *management.StudentService
	leaves       *management.LeaveService
	gates        *authzhttp.Gates
	log          *zap.Logger
}

// NewHandler wires the management HTTP surface.
func NewHandler(
	colleges *management.CollegeService,
	departments *management.DepartmentService,
	programs *management.ProgramService,
	years *management.AcademicYearService,
	semesters *management.SemesterService,
	curriculum *management.CurriculumService,
	settings *management.SettingsService,
	applications *management.ApplicationService,
	enrollment *management.EnrollmentService,
	cohortGroups *management.CohortGroupService,
	requests *management.RequestService,
	courses *management.CourseService,
	offerings *management.OfferingService,
	teachers *management.TeacherService,
	students *management.StudentService,
	leaves *management.LeaveService,
	gates *authzhttp.Gates,
	log *zap.Logger,
) *Handler {
	return &Handler{
		colleges:     colleges,
		departments:  departments,
		programs:     programs,
		years:        years,
		semesters:    semesters,
		curriculum:   curriculum,
		settings:     settings,
		applications: applications,
		enrollment:   enrollment,
		cohortGroups: cohortGroups,
		requests:     requests,
		courses:      courses,
		offerings:    offerings,
		teachers:     teachers,
		students:     students,
		leaves:       leaves,
		gates:        gates,
		log:          log,
	}
}

// Routes maps all management endpoints. public is the unauthenticated
// directory; every protected group below mounts its own gate. The /me
// subtree and application/enrollment-request submission are self-scoped —
// the caller acts on their own records, so no gate applies.
func (h *Handler) Routes(public, protected *gin.RouterGroup) {
	// ── Public directory reads ────────────────────────────────────────────────
	public.GET("/colleges", h.GetPublicColleges)
	public.GET("/colleges/:id", h.GetPublicCollege)
	public.GET("/colleges/:id/departments", h.GetPublicDepartments)
	public.GET("/departments/:id", h.GetPublicDepartment)
	public.GET("/departments/:id/programs", h.GetPublicPrograms)
	public.GET("/about", h.GetPublicAbout)

	// ── Self-scoped (no gate) ─────────────────────────────────────────────────
	protected.POST("/applications", h.CreateApplication)
	protected.GET("/me/applications", h.ListMyApplications)
	protected.GET("/me/applications/:id", h.GetMyApplication)
	protected.PUT("/me/applications/:id", h.UpdateMyApplication)
	protected.PUT("/me/applications/:id/withdraw", h.WithdrawApplication)
	protected.GET("/me/enrollments", h.GetMyEnrollments)
	protected.POST("/enrollment-requests/pretake", h.CreatePretake)
	protected.POST("/enrollment-requests/retake", h.CreateRetake)
	protected.GET("/me/enrollment-requests", h.GetMyEnrollmentRequests)
	protected.GET("/me/teachings", h.GetMyTeachingOfferings)
	protected.GET("/me/student", h.GetMyStudentRecord)
	protected.GET("/me/transcript", h.GetMyTranscript)
	protected.GET("/offerings/:offeringId/access-level", h.GetAccessLevel)

	// ── University structure ──────────────────────────────────────────────────
	colleges := protected.Group("/colleges")
	h.gates.Staff(colleges, authz.ResourceCollege)
	colleges.GET("", h.ListColleges)
	colleges.POST("", h.CreateCollege)
	colleges.GET("/:id", h.GetCollege)
	colleges.PUT("/:id", h.UpdateCollege)
	colleges.GET("/:id/departments", h.ListDepartments)

	departments := protected.Group("/departments")
	h.gates.Staff(departments, authz.ResourceDepartment)
	departments.GET("", h.ListDepartments)
	departments.POST("", h.CreateDepartment) // narrows to the body's college
	departments.GET("/:id", h.GetDepartment)
	departments.PUT("/:id", h.UpdateDepartment)
	departments.GET("/:id/programs", h.ListPrograms)

	programs := protected.Group("/programs")
	h.gates.Staff(programs, authz.ResourceProgram)
	programs.GET("", h.ListPrograms)
	programs.POST("", h.CreateProgram) // narrows to the body's department
	programs.GET("/:id", h.GetProgram)
	programs.PUT("/:id", h.UpdateProgram)
	programs.GET("/:id/cohort-groups", h.ListCohortGroups)
	programs.GET("/:id/cohorts", h.ListCohortYears)

	// ── Curriculum (rows top-level, collections under their program) ──────────
	curriculum := protected.Group("/programs/:id/curriculum")
	h.gates.StaffUnder(curriculum, authz.ResourceCurriculum, authz.ResourceProgram, "id")
	curriculum.GET("", h.ListCurriculum)
	curriculum.POST("", h.AddToCurriculum)

	requirements := protected.Group("/programs/:id/requirements")
	h.gates.StaffUnder(requirements, authz.ResourceCurriculum, authz.ResourceProgram, "id")
	requirements.GET("", h.ListRequirements)
	requirements.POST("", h.SetRequirement)

	curriculumRows := protected.Group("/curriculum")
	h.gates.Staff(curriculumRows, authz.ResourceCurriculum)
	curriculumRows.DELETE("/:id", h.RemoveFromCurriculum)

	// ── Academic calendar ─────────────────────────────────────────────────────
	years := protected.Group("/academic-years")
	h.gates.Staff(years, authz.ResourceAcademicYear)
	years.GET("", h.ListAcademicYears)
	years.POST("", h.CreateAcademicYear)
	years.GET("/:id", h.GetAcademicYear)
	years.PUT("/:id", h.UpdateAcademicYear)

	semesters := protected.Group("/semesters")
	h.gates.Staff(semesters, authz.ResourceSemester)
	semesters.GET("", h.ListSemesters)
	semesters.POST("", h.CreateSemester)
	semesters.GET("/:id", h.GetSemester)
	semesters.PUT("/:id", h.UpdateSemester)
	semesters.DELETE("/:id", h.DeleteSemester)
	semesters.PUT("/:id/status", h.UpdateSemesterStatus)
	semesters.POST("/:id", h.SemesterCustom) // :definalize, :generateOfferings, :bulkEnroll, :end

	// ── Settings (singleton) ──────────────────────────────────────────────────
	settings := protected.Group("/settings")
	h.gates.StaffSingleton(settings, authz.ResourceSettings)
	settings.GET("", h.GetSettings)
	settings.PUT("", h.UpdateSettings)
	settings.GET("/institution", h.GetInstitution)
	settings.PUT("/institution", h.UpdateInstitution)
	settings.GET("/features", h.GetFeatures)
	settings.PUT("/features", h.UpdateFeatures)

	// ── Applications (admin; submission is self-scoped above) ────────────────
	applications := protected.Group("/applications")
	h.gates.Staff(applications, authz.ResourceApplication)
	applications.GET("", h.ListApplications)
	applications.GET("/:id", h.GetApplication)
	applications.POST("/:id", h.ApplicationCustom) // :review

	// ── Offerings (":offeringId": classroom owns the param name) ─────────────
	offerings := protected.Group("/offerings")
	h.gates.StaffAt(offerings, authz.ResourceOffering, "offeringId")
	offerings.GET("", h.ListOfferings)
	offerings.POST("", h.CreateOffering) // narrows to the body's course
	offerings.GET("/:offeringId", h.GetOffering)
	offerings.PUT("/:offeringId", h.UpdateOffering)
	offerings.DELETE("/:offeringId", h.DeleteOffering)

	enrollments := protected.Group("/offerings/:offeringId/enrollments")
	h.gates.StaffUnder(enrollments, authz.ResourceEnrollment, authz.ResourceOffering, "offeringId")
	enrollments.GET("", h.ListEnrollments)
	enrollments.POST("", h.EnrollStudent)
	enrollments.DELETE("/:student_id", h.DropEnrollment)

	teachers := protected.Group("/offerings/:offeringId/teachers")
	h.gates.StaffUnder(teachers, authz.ResourceTeacher, authz.ResourceOffering, "offeringId")
	teachers.GET("", h.ListTeachers)
	teachers.POST("", h.AddTeacher)
	teachers.DELETE("/:user_id", h.RemoveTeacher)
	teachers.PATCH("/:user_id", h.UpdateTeacherRole)

	// ── Cohort groups (rank-gated; program-scoped reads live under /programs) ─
	cohortGroups := protected.Group("/cohort-groups")
	h.gates.Staff(cohortGroups, authz.ResourceCohortGroup)
	cohortGroups.POST("", h.CreateCohortGroup)
	cohortGroups.POST("/assign", h.AssignToCohortGroup)
	cohortGroups.DELETE("/:id/members/:student_id", h.RemoveFromCohortGroup)

	// ── Enrollment requests (admin; student requests are self-scoped above) ──
	requests := protected.Group("/enrollment-requests")
	h.gates.Staff(requests, authz.ResourceEnrollment)
	requests.GET("", h.ListEnrollmentRequests)
	requests.GET("/:id", h.GetEnrollmentRequest)
	requests.POST("/:id", h.RequestCustom) // :approve, :reject

	// ── Courses ───────────────────────────────────────────────────────────────
	courses := protected.Group("/courses")
	h.gates.Staff(courses, authz.ResourceCourse)
	courses.GET("", h.ListCourses)
	courses.POST("", h.CreateCourse) // narrows to the body's department
	courses.GET("/:id", h.GetCourse)
	courses.PUT("/:id", h.UpdateCourse)
	courses.DELETE("/:id", h.DeleteCourse)
	courses.GET("/:id/siblings", h.GetSiblingCourses)

	// ── Students and leaves ───────────────────────────────────────────────────
	students := protected.Group("/students")
	h.gates.Staff(students, authz.ResourceStudent)
	students.GET("", h.ListStudents)
	students.POST("", h.CreateStudent) // narrows to the body's program
	students.GET("/:id", h.GetStudent)
	students.PUT("/:id", h.UpdateStudent)
	students.PUT("/:id/status", h.UpdateStudentStatus)
	students.GET("/:id/transcript", h.GetTranscript)
	students.GET("/:id/leaves", h.ListLeaves)
	students.GET("/:id/history", h.ListCohortHistory)
	students.POST("/:id", h.StudentCustom) // :requestLeave

	leaves := protected.Group("/leaves")
	h.gates.Staff(leaves, authz.ResourceStudent)
	leaves.POST("/:id", h.LeaveCustom) // :approve, :end
}

// respondError is the context's single error→status translation point (the
// second of the two sanctioned translation points): every endpoint funnels
// its service errors through here. Unknown errors are logged and surface as
// an opaque 500.
func (h *Handler) respondError(c *gin.Context, err error) {
	switch {
	// Not found.
	case errors.Is(err, management.ErrCollegeNotFound),
		errors.Is(err, management.ErrDepartmentNotFound),
		errors.Is(err, management.ErrProgramNotFound),
		errors.Is(err, management.ErrAcademicYearNotFound),
		errors.Is(err, management.ErrSemesterNotFound),
		errors.Is(err, management.ErrCurriculumNotFound),
		errors.Is(err, management.ErrRequirementNotFound),
		errors.Is(err, management.ErrCourseNotFound),
		errors.Is(err, management.ErrOfferingNotFound),
		errors.Is(err, management.ErrTeacherNotFound),
		errors.Is(err, management.ErrCohortGroupNotFound),
		errors.Is(err, management.ErrEnrollmentNotFound),
		errors.Is(err, management.ErrRequestNotFound),
		errors.Is(err, management.ErrApplicationNotFound),
		errors.Is(err, management.ErrStudentNotFound),
		errors.Is(err, management.ErrLeaveNotFound),
		errors.Is(err, management.ErrUserNotFound),
		errors.Is(err, management.ErrSettingsNotFound):
		response.NotFound(c, err.Error())

	// Conflicts: duplicates, lost CAS races, and already-decided reviews.
	case errors.Is(err, management.ErrConflict),
		errors.Is(err, management.ErrSettingsConflict),
		errors.Is(err, management.ErrCodeExists),
		errors.Is(err, management.ErrDuplicateYear),
		errors.Is(err, management.ErrDuplicateSemester),
		errors.Is(err, management.ErrDuplicateCurriculum),
		errors.Is(err, management.ErrDuplicateCode),
		errors.Is(err, management.ErrDuplicateOffering),
		errors.Is(err, management.ErrDuplicateApplication),
		errors.Is(err, management.ErrDuplicateStudent),
		errors.Is(err, management.ErrDuplicateRequest),
		errors.Is(err, management.ErrDuplicateCohortGroup),
		errors.Is(err, management.ErrAlreadyEnrolled),
		errors.Is(err, management.ErrAlreadyTeacher),
		errors.Is(err, management.ErrAlreadyReviewed),
		errors.Is(err, management.ErrAlreadyOnLeave),
		errors.Is(err, management.ErrLeaveAlreadyApproved):
		response.Conflict(c, err.Error())

	// Invalid state for the requested transition.
	case errors.Is(err, management.ErrInvalidStatusTransition),
		errors.Is(err, management.ErrSemesterNotFinalized),
		errors.Is(err, management.ErrSemesterArchived),
		errors.Is(err, management.ErrSemesterNotActive),
		errors.Is(err, management.ErrOfferingsNotFinalized),
		errors.Is(err, management.ErrApplicationCannotUpdate),
		errors.Is(err, management.ErrApplicationCannotWithdraw),
		errors.Is(err, management.ErrApplicationCannotReview),
		errors.Is(err, management.ErrLeaveEnded),
		errors.Is(err, management.ErrNotOnLeave):
		response.Conflict(c, err.Error())

	// Limits.
	case errors.Is(err, management.ErrCollegeLimitReached),
		errors.Is(err, management.ErrDepartmentLimitReached),
		errors.Is(err, management.ErrProgramLimitReached):
		response.Forbidden(c, err.Error())

	// Ownership.
	case errors.Is(err, management.ErrApplicationAccessDenied),
		errors.Is(err, management.ErrApplicationCannotReviewOwn):
		response.Forbidden(c, err.Error())

	// Bad input or business-rule refusal.
	case errors.Is(err, management.ErrInvalidStatus),
		errors.Is(err, management.ErrInvalidLeaveType),
		errors.Is(err, management.ErrInvalidRequestType),
		errors.Is(err, management.ErrMissingInstitutionName),
		errors.Is(err, management.ErrInvalidGradingDisplay),
		errors.Is(err, management.ErrInvalidSemestersPerYear),
		errors.Is(err, management.ErrProgramInactive),
		errors.Is(err, management.ErrAgeTooYoung),
		errors.Is(err, management.ErrAgeTooOld),
		errors.Is(err, management.ErrNoPrerequisite),
		errors.Is(err, management.ErrPrerequisitePassed),
		errors.Is(err, management.ErrCourseNotFailed),
		errors.Is(err, management.ErrNotNaturalCohort),
		errors.Is(err, management.ErrNotEnrolled):
		response.BadRequest(c, err.Error())

	default:
		h.log.Error("management handler error", zap.Error(err))
		response.InternalError(c)
	}
}

// scopeFrom translates the gate's row constraint into the domain's filter
// vocabulary: narrow-scoped staff list only their own unit's rows.
func scopeFrom(c *gin.Context) management.ScopeFilter {
	f := authzhttp.Access(c).Filter()
	return management.ScopeFilter{ProgramID: f.ProgramID, DepartmentID: f.DepartmentID, CollegeID: f.CollegeID}
}
