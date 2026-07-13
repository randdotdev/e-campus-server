package authz

// The factory policy table: compiled-in data, seeded on boot for unknown
// pairs, re-flashed wholesale by Reset. Deliberately absent: the 'policy'
// resource — policy administration is gated by a hardcoded super-admin
// check, never by stored rows. Entries stay explicit and greppable even
// where a wider scope already implies coverage.

// DefaultPolicies returns the compiled-in policy map. Callers must treat it
// as read-only.
func DefaultPolicies() map[PolicyKey]Policy { return defaultPolicies }

var defaultPolicies = map[PolicyKey]Policy{
	// ── structural resources ────────────────────────────────────────────
	{ResourceDepartment, ActionGet}: {Staff: []StaffPermission{
		{MinLevel: LevelViewer, Scope: ScopeDepartment},
		{MinLevel: LevelViewer, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceDepartment, ActionUpdate}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeDepartment},
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceCollege, ActionGet}: {Staff: []StaffPermission{
		{MinLevel: LevelViewer, Scope: ScopeCollege},
		{MinLevel: LevelViewer, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceCollege, ActionUpdate}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeCollege},
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceProgram, ActionGet}: {Staff: []StaffPermission{
		{MinLevel: LevelViewer, Scope: ScopeProgram},
		{MinLevel: LevelViewer, Scope: ScopeDepartment},
		{MinLevel: LevelViewer, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceProgram, ActionUpdate}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeProgram},
		{MinLevel: LevelAdmin, Scope: ScopeDepartment},
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceCollege, ActionList}: {Staff: []StaffPermission{
		{MinLevel: LevelViewer, Scope: ScopeCollege},
		{MinLevel: LevelViewer, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceCollege, ActionCreate}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceDepartment, ActionList}: {Staff: []StaffPermission{
		{MinLevel: LevelViewer, Scope: ScopeDepartment},
		{MinLevel: LevelViewer, Scope: ScopeCollege},
		{MinLevel: LevelViewer, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceDepartment, ActionCreate}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeCollege},
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceProgram, ActionList}: {Staff: []StaffPermission{
		{MinLevel: LevelViewer, Scope: ScopeProgram},
		{MinLevel: LevelViewer, Scope: ScopeDepartment},
		{MinLevel: LevelViewer, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceProgram, ActionCreate}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeDepartment},
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceUniversity, ActionGet}: {Staff: []StaffPermission{
		{MinLevel: LevelViewer, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceUniversity, ActionUpdate}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourcePlatform, ActionGet}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourcePlatform, ActionUpdate}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},

	// ── academic calendar ───────────────────────────────────────────────
	{ResourceAcademicYear, ActionGet}: {Staff: []StaffPermission{
		{MinLevel: LevelViewer, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceAcademicYear, ActionList}: {Staff: []StaffPermission{
		{MinLevel: LevelViewer, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceAcademicYear, ActionCreate}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceAcademicYear, ActionUpdate}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceAcademicYear, ActionDelete}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceSemester, ActionGet}: {Staff: []StaffPermission{
		{MinLevel: LevelViewer, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceSemester, ActionList}: {Staff: []StaffPermission{
		{MinLevel: LevelViewer, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceSemester, ActionCreate}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceSemester, ActionUpdate}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceSemester, ActionDelete}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	// Semester lifecycle custom methods: same custodians as update.
	{ResourceSemester, ActionActivate}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceSemester, ActionStartGrading}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceSemester, ActionFinalize}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceSemester, ActionDefinalize}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceSemester, ActionGenerateOfferings}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceSemester, ActionBulkEnroll}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceSemester, ActionEnd}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},

	// ── course catalogue ────────────────────────────────────────────────
	// Seat permissions live on the offering entries, never here: a course
	// is only the catalogue entity; classrooms live on offerings.
	{ResourceCourse, ActionGet}: {Staff: []StaffPermission{
		{MinLevel: LevelViewer, Scope: ScopeProgram},
		{MinLevel: LevelViewer, Scope: ScopeDepartment},
		{MinLevel: LevelViewer, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceCourse, ActionList}: {Staff: []StaffPermission{
		{MinLevel: LevelViewer, Scope: ScopeProgram},
		{MinLevel: LevelViewer, Scope: ScopeDepartment},
		{MinLevel: LevelViewer, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceCourse, ActionUpdate}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeDepartment},
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceCourse, ActionCreate}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeDepartment},
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceCourse, ActionDelete}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeDepartment},
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},

	// ── offering lifecycle ──────────────────────────────────────────────
	{ResourceOffering, ActionCreate}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeDepartment},
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceOffering, ActionGet}: {
		Staff: []StaffPermission{
			{MinLevel: LevelViewer, Scope: ScopeDepartment},
			{MinLevel: LevelViewer, Scope: ScopeUniversity},
			{MinLevel: LevelAdmin, Scope: ScopePlatform},
		},
		Offering: []OfferingRole{OfferingRoleTeacher, OfferingRoleAssistant, OfferingRoleStudent, OfferingRoleObserver},
	},
	{ResourceOffering, ActionList}: {
		Staff: []StaffPermission{
			{MinLevel: LevelAdmin, Scope: ScopeDepartment},
			{MinLevel: LevelAdmin, Scope: ScopeCollege},
			{MinLevel: LevelAdmin, Scope: ScopeUniversity},
			{MinLevel: LevelAdmin, Scope: ScopePlatform},
		},
		Offering: []OfferingRole{OfferingRoleTeacher, OfferingRoleAssistant, OfferingRoleStudent, OfferingRoleObserver},
	},
	{ResourceOffering, ActionUpdate}: {
		Staff: []StaffPermission{
			{MinLevel: LevelAdmin, Scope: ScopeDepartment},
			{MinLevel: LevelAdmin, Scope: ScopeCollege},
			{MinLevel: LevelAdmin, Scope: ScopeUniversity},
			{MinLevel: LevelAdmin, Scope: ScopePlatform},
		},
		Offering: []OfferingRole{OfferingRoleTeacher, OfferingRoleAssistant},
	},
	{ResourceOffering, ActionDelete}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeDepartment},
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},

	// ── teaching assignments (course_teachers, scoped via the offering) ─
	{ResourceTeacher, ActionList}: {Staff: []StaffPermission{
		{MinLevel: LevelViewer, Scope: ScopeDepartment},
		{MinLevel: LevelViewer, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceTeacher, ActionCreate}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeDepartment},
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceTeacher, ActionUpdate}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeDepartment},
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceTeacher, ActionDelete}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeDepartment},
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},

	// ── enrollment ──────────────────────────────────────────────────────
	{ResourceEnrollment, ActionCreate}: {
		Staff: []StaffPermission{
			{MinLevel: LevelAdmin, Scope: ScopeDepartment},
			{MinLevel: LevelAdmin, Scope: ScopeUniversity},
			{MinLevel: LevelAdmin, Scope: ScopePlatform},
		},
		Offering: []OfferingRole{OfferingRoleTeacher, OfferingRoleAssistant},
	},
	{ResourceEnrollment, ActionUpdate}: {
		Staff: []StaffPermission{
			{MinLevel: LevelAdmin, Scope: ScopeDepartment},
			{MinLevel: LevelAdmin, Scope: ScopeCollege},
			{MinLevel: LevelAdmin, Scope: ScopeUniversity},
			{MinLevel: LevelAdmin, Scope: ScopePlatform},
		},
		Offering: []OfferingRole{OfferingRoleTeacher, OfferingRoleAssistant},
	},
	{ResourceEnrollment, ActionDelete}: {
		Staff: []StaffPermission{
			{MinLevel: LevelAdmin, Scope: ScopeDepartment},
			{MinLevel: LevelAdmin, Scope: ScopeUniversity},
			{MinLevel: LevelAdmin, Scope: ScopePlatform},
		},
		Offering: []OfferingRole{OfferingRoleTeacher, OfferingRoleAssistant},
	},
	{ResourceEnrollment, ActionList}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceEnrollment, ActionGet}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	// Enrollment-request decisions.
	{ResourceEnrollment, ActionApprove}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceEnrollment, ActionReject}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},

	// ── user management ─────────────────────────────────────────────────
	{ResourceUser, ActionGet}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceUser, ActionList}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceUser, ActionCreate}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceUser, ActionUpdate}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceUser, ActionDelete}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},

	// ── students ────────────────────────────────────────────────────────
	{ResourceStudent, ActionGet}: {Staff: []StaffPermission{
		{MinLevel: LevelViewer, Scope: ScopeProgram},
		{MinLevel: LevelViewer, Scope: ScopeDepartment},
		{MinLevel: LevelViewer, Scope: ScopeCollege},
		{MinLevel: LevelViewer, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceStudent, ActionList}: {Staff: []StaffPermission{
		{MinLevel: LevelViewer, Scope: ScopeDepartment},
		{MinLevel: LevelViewer, Scope: ScopeCollege},
		{MinLevel: LevelViewer, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceStudent, ActionCreate}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeProgram},
		{MinLevel: LevelAdmin, Scope: ScopeDepartment},
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceStudent, ActionUpdate}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeProgram},
		{MinLevel: LevelAdmin, Scope: ScopeDepartment},
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceStudent, ActionDelete}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	// Leave lifecycle: opening one follows student update (program staff
	// manage their own students); deciding and ending are university-level
	// registrar work, rank-only — leave rows carry no lineage yet.
	{ResourceStudent, ActionRequestLeave}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeProgram},
		{MinLevel: LevelAdmin, Scope: ScopeDepartment},
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceStudent, ActionApprove}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceStudent, ActionEnd}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},

	// ── cohort groups (rank-only: no cohort_group lineage yet) ──────────
	{ResourceCohortGroup, ActionList}: {Staff: []StaffPermission{
		{MinLevel: LevelViewer, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceCohortGroup, ActionGet}: {Staff: []StaffPermission{
		{MinLevel: LevelViewer, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceCohortGroup, ActionCreate}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceCohortGroup, ActionUpdate}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceCohortGroup, ActionDelete}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},

	// ── admissions ──────────────────────────────────────────────────────
	{ResourceApplication, ActionList}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeProgram},
		{MinLevel: LevelAdmin, Scope: ScopeDepartment},
		{MinLevel: LevelAdmin, Scope: ScopeCollege},
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceApplication, ActionGet}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeProgram},
		{MinLevel: LevelAdmin, Scope: ScopeDepartment},
		{MinLevel: LevelAdmin, Scope: ScopeCollege},
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceApplication, ActionUpdate}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeProgram},
		{MinLevel: LevelAdmin, Scope: ScopeDepartment},
		{MinLevel: LevelAdmin, Scope: ScopeCollege},
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceApplication, ActionReview}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeProgram},
		{MinLevel: LevelAdmin, Scope: ScopeDepartment},
		{MinLevel: LevelAdmin, Scope: ScopeCollege},
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},

	// ── announcements ───────────────────────────────────────────────────
	// Course-scoped posts are the classroom feed: teaching seats post.
	{ResourcePost, ActionCreate}: {
		Staff: []StaffPermission{
			{MinLevel: LevelAdmin, Scope: ScopeProgram},
			{MinLevel: LevelAdmin, Scope: ScopeDepartment},
			{MinLevel: LevelAdmin, Scope: ScopeCollege},
			{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		},
		Offering: []OfferingRole{OfferingRoleTeacher, OfferingRoleAssistant},
	},
	{ResourceActivity, ActionCreate}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeDepartment},
		{MinLevel: LevelAdmin, Scope: ScopeCollege},
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
	}},
	// Post authority (CheckPost's policy arms): who may act on someone
	// else's post in a scope they govern. Authors act on their own posts
	// through the owner arm; plain members read published content through
	// the member arm — neither needs a row here.
	{ResourcePost, ActionGet}:    {Staff: postStaff, Offering: teachingSeats},
	{ResourcePost, ActionList}:   {Staff: postStaff, Offering: teachingSeats},
	{ResourcePost, ActionUpdate}: {Staff: postStaff, Offering: teachingSeats},
	{ResourcePost, ActionDelete}: {Staff: postStaff, Offering: teachingSeats},
	{ResourcePost, ActionPin}:    {Staff: postStaff, Offering: teachingSeats},
	{ResourcePost, ActionAttach}: {Staff: postStaff, Offering: teachingSeats},

	// ── classroom (2026-07-06: rebuilt with the classroom migration) ────
	// Teaching seats run their offering; student seats act on their own
	// work through the custom methods; department/university staff read,
	// admins intervene. allSeats/teachingSeats/studentSeat below the map.

	// content: the material tree.
	{ResourceContent, ActionGet}:        {Staff: viewerStaff, Offering: allSeats},
	{ResourceContent, ActionList}:       {Staff: viewerStaff, Offering: allSeats},
	{ResourceContent, ActionCreate}:     {Staff: adminStaff, Offering: teachingSeats},
	{ResourceContent, ActionUpdate}:     {Staff: adminStaff, Offering: teachingSeats},
	{ResourceContent, ActionDelete}:     {Staff: adminStaff, Offering: teachingSeats},
	{ResourceContent, ActionAttach}:     {Offering: teachingSeats},
	{ResourceContent, ActionSchedule}:   {Offering: teachingSeats},
	{ResourceContent, ActionUnschedule}: {Offering: teachingSeats},

	// assignments and their submissions.
	{ResourceAssignment, ActionGet}:     {Staff: viewerStaff, Offering: allSeats},
	{ResourceAssignment, ActionList}:    {Staff: viewerStaff, Offering: allSeats},
	{ResourceAssignment, ActionCreate}:  {Staff: adminStaff, Offering: teachingSeats},
	{ResourceAssignment, ActionUpdate}:  {Staff: adminStaff, Offering: teachingSeats},
	{ResourceAssignment, ActionDelete}:  {Staff: adminStaff, Offering: []OfferingRole{OfferingRoleTeacher}},
	{ResourceAssignment, ActionAttach}:  {Offering: teachingSeats},
	{ResourceAssignment, ActionSave}:    {Offering: studentSeat},
	{ResourceAssignment, ActionSubmit}:  {Offering: studentSeat},
	{ResourceAssignment, ActionDiscard}: {Offering: studentSeat},
	{ResourceAssignment, ActionGrade}:   {Offering: teachingSeats},

	// exams, their attempts, and the question bank (bank rides the exam
	// resource: same custodians, same sensitivity).
	{ResourceExam, ActionGet}:     {Staff: viewerStaff, Offering: allSeats},
	{ResourceExam, ActionList}:    {Staff: viewerStaff, Offering: allSeats},
	{ResourceExam, ActionCreate}:  {Staff: adminStaff, Offering: teachingSeats},
	{ResourceExam, ActionUpdate}:  {Offering: teachingSeats},
	{ResourceExam, ActionDelete}:  {Offering: []OfferingRole{OfferingRoleTeacher}},
	{ResourceExam, ActionPublish}: {Offering: teachingSeats},
	{ResourceExam, ActionClose}:   {Offering: []OfferingRole{OfferingRoleTeacher}},
	{ResourceExam, ActionStart}:   {Offering: studentSeat},
	{ResourceExam, ActionSave}:    {Offering: studentSeat},
	{ResourceExam, ActionSubmit}:  {Offering: studentSeat},
	{ResourceExam, ActionGrade}:   {Offering: teachingSeats},
	{ResourceExam, ActionReview}:  {Offering: teachingSeats},
	{ResourceExam, ActionRecord}:  {Offering: teachingSeats},

	// attendance and excuses.
	{ResourceAttendance, ActionGet}:        {Staff: viewerStaff, Offering: allSeats},
	{ResourceAttendance, ActionList}:       {Staff: viewerStaff, Offering: []OfferingRole{OfferingRoleTeacher, OfferingRoleAssistant, OfferingRoleStudent}},
	{ResourceAttendance, ActionUpdate}:     {Offering: teachingSeats},
	{ResourceAttendance, ActionInitialize}: {Offering: teachingSeats},
	{ResourceAttendance, ActionMark}:       {Offering: teachingSeats},
	{ResourceAttendance, ActionExcuse}:     {Offering: studentSeat},
	{ResourceAttendance, ActionReview}:     {Offering: teachingSeats},

	// final grades and the rule set.
	{ResourceGrade, ActionGet}:    {Offering: allSeats},
	{ResourceGrade, ActionList}:   {Staff: viewerStaff, Offering: []OfferingRole{OfferingRoleTeacher, OfferingRoleAssistant, OfferingRoleStudent}},
	{ResourceGrade, ActionCreate}: {Offering: teachingSeats},
	{ResourceGrade, ActionUpdate}: {Offering: teachingSeats},

	// questions & answers.
	{ResourceQA, ActionCreate}: {Offering: allSeats},
	{ResourceQA, ActionGet}:    {Staff: viewerStaff, Offering: allSeats},
	{ResourceQA, ActionList}:   {Staff: viewerStaff, Offering: allSeats},
	{ResourceQA, ActionUpdate}: {Staff: adminStaff, Offering: []OfferingRole{OfferingRoleTeacher, OfferingRoleAssistant, OfferingRoleStudent}},
	{ResourceQA, ActionDelete}: {Offering: []OfferingRole{OfferingRoleTeacher, OfferingRoleAssistant, OfferingRoleStudent}},
	{ResourceQA, ActionAnswer}: {Offering: teachingSeats},
	{ResourceQA, ActionReject}: {Offering: teachingSeats},

	// projects: teacher-run, team-submitted.
	{ResourceProject, ActionGet}:        {Staff: viewerStaff, Offering: allSeats},
	{ResourceProject, ActionList}:       {Staff: viewerStaff, Offering: allSeats},
	{ResourceProject, ActionCreate}:     {Offering: []OfferingRole{OfferingRoleTeacher}},
	{ResourceProject, ActionUpdate}:     {Offering: teachingSeats},
	{ResourceProject, ActionDelete}:     {Offering: []OfferingRole{OfferingRoleTeacher}},
	{ResourceProject, ActionAttach}:     {Offering: teachingSeats},
	{ResourceProject, ActionRegister}:   {Offering: studentSeat},
	{ResourceProject, ActionUnregister}: {Offering: studentSeat},
	{ResourceProject, ActionFormGroups}: {Offering: teachingSeats},
	{ResourceProject, ActionSave}:       {Offering: studentSeat},
	{ResourceProject, ActionSubmit}:     {Offering: studentSeat},
	{ResourceProject, ActionGrade}:      {Offering: teachingSeats},

	// ── moderation (mutes) ──────────────────────────────────────────────
	// Offering-scoped muting: a teaching seat silences a participant in its
	// own class; department staff and up moderate across their scope.
	// University-wide mutes are not here — they run on ResourceUser.
	{ResourceMute, ActionList}:   {Staff: viewerStaff, Offering: teachingSeats},
	{ResourceMute, ActionCreate}: {Staff: adminStaff, Offering: teachingSeats},
	{ResourceMute, ActionDelete}: {Staff: adminStaff, Offering: teachingSeats},

	// ── curriculum, settings, billing ───────────────────────────────────
	{ResourceCurriculum, ActionList}: {Staff: []StaffPermission{
		{MinLevel: LevelViewer, Scope: ScopeProgram},
		{MinLevel: LevelViewer, Scope: ScopeDepartment},
		{MinLevel: LevelViewer, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceCurriculum, ActionCreate}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeProgram},
		{MinLevel: LevelAdmin, Scope: ScopeDepartment},
		{MinLevel: LevelAdmin, Scope: ScopeCollege},
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceCurriculum, ActionUpdate}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeProgram},
		{MinLevel: LevelAdmin, Scope: ScopeDepartment},
		{MinLevel: LevelAdmin, Scope: ScopeCollege},
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceCurriculum, ActionDelete}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeProgram},
		{MinLevel: LevelAdmin, Scope: ScopeDepartment},
		{MinLevel: LevelAdmin, Scope: ScopeCollege},
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceSettings, ActionGet}: {Staff: []StaffPermission{
		{MinLevel: LevelViewer, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceSettings, ActionUpdate}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceSubscription, ActionGet}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
	{ResourceSubscription, ActionUpdate}: {Staff: []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}},
}

// Seat shorthands for the classroom block; a slice per audience keeps the
// table scannable without hiding which seats an entry admits.
var (
	allSeats      = []OfferingRole{OfferingRoleTeacher, OfferingRoleAssistant, OfferingRoleStudent, OfferingRoleObserver}
	teachingSeats = []OfferingRole{OfferingRoleTeacher, OfferingRoleAssistant}
	studentSeat   = []OfferingRole{OfferingRoleStudent}

	postStaff = []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeProgram},
		{MinLevel: LevelAdmin, Scope: ScopeDepartment},
		{MinLevel: LevelAdmin, Scope: ScopeCollege},
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}
	viewerStaff = []StaffPermission{
		{MinLevel: LevelViewer, Scope: ScopeDepartment},
		{MinLevel: LevelViewer, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}
	adminStaff = []StaffPermission{
		{MinLevel: LevelAdmin, Scope: ScopeDepartment},
		{MinLevel: LevelAdmin, Scope: ScopeCollege},
		{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		{MinLevel: LevelAdmin, Scope: ScopePlatform},
	}
)
