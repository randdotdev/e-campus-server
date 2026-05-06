package authz

const (
	SuperAdmin = "super_admin"
	Admin      = "admin"
	Operator   = "operator"
	Viewer     = "viewer"
)

const (
	ScopePlatform   = "platform"
	ScopeUniversity = "university"
	ScopeCollege    = "college"
	ScopeDepartment = "department"
	ScopeProgram    = "program"
)

const (
	DomainAdministration = "administration"
	DomainAccountant     = "accountant"
	DomainRegistrar      = "registrar"
	DomainScheduler      = "scheduler"
	DomainAdmissions     = "admissions"
	DomainHR             = "hr"
)

const (
	CourseRoleTeacher   = "teacher"
	CourseRoleAssistant = "assistant"
	CourseRoleStudent   = "student"
	CourseRoleObserver  = "observer"
)

const ResourceTypeOffering = "offering"

var permissionRank = map[string]int{
	SuperAdmin: 4,
	Admin:      3,
	Operator:   2,
	Viewer:     1,
}

var scopeRank = map[string]int{
	ScopePlatform:   5,
	ScopeUniversity: 4,
	ScopeCollege:    3,
	ScopeDepartment: 2,
	ScopeProgram:    1,
}

type Entity string

const (
	ResourceCourse       Entity = "course"
	ResourceOffering     Entity = "offering"
	ResourceStudent      Entity = "student"
	ResourceExam         Entity = "exam"
	ResourceAssignment   Entity = "assignment"
	ResourceAcademicYear Entity = "academic_year"
	ResourceSemester     Entity = "semester"
	ResourceEnrollment   Entity = "enrollment"
	ResourceGrade        Entity = "grade"
	ResourceAttendance   Entity = "attendance"
	ResourceUser         Entity = "user"
	ResourceDepartment   Entity = "department"
	ResourceCollege      Entity = "college"
	ResourceProgram      Entity = "program"
	ResourceUniversity   Entity = "university"
	ResourcePolicy       Entity = "policy"
	ResourceNews         Entity = "news"
	ResourcePost         Entity = "post"
	ResourceQA           Entity = "qa"
	ResourceSettings     Entity = "settings"
	ResourceApplication  Entity = "application"
	ResourceCohortGroup  Entity = "cohort_group"
	ResourceSubscription Entity = "subscription"
	ResourceCurriculum   Entity = "curriculum"
	ResourceProject      Entity = "project"
)

const (
	ActionGet    = "get"
	ActionList   = "list"
	ActionCreate = "create"
	ActionUpdate = "update"
	ActionDelete = "delete"
	ActionSubmit = "submit"
)
