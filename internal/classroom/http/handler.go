// Package http is the HTTP transport for the classroom domain. Every
// offering-scoped route lives under /offerings/:offeringId behind the
// classroom gate; custom methods ride the URL's colon suffix
// (POST /exams/:id:publish). Teams are offering-free and student-managed,
// so they mount on the plain authenticated group — their authority is
// structural (leader, member) and enforced in the domain.
package http

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/randdotdev/e-campus-server/internal/authz"
	authzhttp "github.com/randdotdev/e-campus-server/internal/authz/http"
	"github.com/randdotdev/e-campus-server/internal/classroom"
	"github.com/randdotdev/e-campus-server/internal/shared/response"
)

// Handler is the classroom context's HTTP surface.
type Handler struct {
	content     *classroom.ContentService
	assignments *classroom.AssignmentService
	questions   *classroom.QuestionService
	exams       *classroom.ExamService
	attendance  *classroom.AttendanceService
	grading     *classroom.GradingService
	qa          *classroom.QAService
	teams       *classroom.TeamService
	projects    *classroom.ProjectService
	log         *zap.Logger
}

func NewHandler(
	content *classroom.ContentService,
	assignments *classroom.AssignmentService,
	questions *classroom.QuestionService,
	exams *classroom.ExamService,
	attendance *classroom.AttendanceService,
	grading *classroom.GradingService,
	qa *classroom.QAService,
	teams *classroom.TeamService,
	projects *classroom.ProjectService,
	log *zap.Logger,
) *Handler {
	return &Handler{
		content: content, assignments: assignments, questions: questions,
		exams: exams, attendance: attendance, grading: grading,
		qa: qa, teams: teams, projects: projects, log: log,
	}
}

// Routes mounts every classroom route. gates guards each offering-scoped
// group with the offering check; rg is the plain authenticated group for
// teams and the student's own calendar.
func (h *Handler) Routes(rg *gin.RouterGroup, gates *authzhttp.Gates) {
	offering := rg.Group("/offerings/:offeringId")

	sections := offering.Group("/sections")
	gates.Classroom(sections, authz.ResourceContent)
	sections.GET("", h.ListSections)
	sections.POST("", h.CreateSection)
	sections.GET("/:id", h.GetSection)
	sections.PATCH("/:id", h.UpdateSection)
	sections.DELETE("/:id", h.DeleteSection)
	sections.GET("/:id/lessons", h.ListLessons)

	lessons := offering.Group("/lessons")
	gates.Classroom(lessons, authz.ResourceContent)
	lessons.POST("", h.CreateLesson)
	lessons.GET("/:id", h.GetLesson)
	lessons.PATCH("/:id", h.UpdateLesson)
	lessons.DELETE("/:id", h.DeleteLesson)
	lessons.POST("/:id", h.LessonCustom) // :attach, :schedule, :unschedule
	lessons.GET("/:id/attachments/:name", h.DownloadLessonAttachment)
	lessons.DELETE("/:id/attachments/:attachmentId", h.DetachLessonFile)

	assignments := offering.Group("/assignments")
	gates.Classroom(assignments, authz.ResourceAssignment)
	assignments.GET("", h.ListAssignments)
	assignments.POST("", h.CreateAssignment)
	assignments.GET("/:id", h.GetAssignment)
	assignments.PATCH("/:id", h.UpdateAssignment)
	assignments.DELETE("/:id", h.DeleteAssignment)
	assignments.POST("/:id", h.AssignmentCustom) // :attach, :save, :submit, :discard, :grade
	assignments.GET("/:id/attachments/:name", h.DownloadAssignmentAttachment)
	assignments.DELETE("/:id/attachments/:attachmentId", h.DetachAssignmentFile)
	assignments.GET("/:id/submission", h.MySubmission)
	assignments.GET("/:id/submissions", h.ListSubmissions)
	assignments.GET("/:id/submissions/:studentId/files/:name", h.DownloadSubmissionFile)

	questions := offering.Group("/questions")
	gates.Classroom(questions, authz.ResourceExam)
	questions.GET("", h.ListQuestions)
	questions.POST("", h.CreateQuestion)
	questions.POST("/bulk", h.BulkCreateQuestions)
	questions.GET("/sample", h.SampleQuestions)
	questions.GET("/:id", h.GetQuestion)
	questions.PATCH("/:id", h.UpdateQuestion)
	questions.DELETE("/:id", h.DeactivateQuestion)

	exams := offering.Group("/exams")
	gates.Classroom(exams, authz.ResourceExam)
	exams.GET("", h.ListExams)
	exams.POST("", h.CreateExam)
	exams.GET("/:id", h.GetExam)
	exams.PATCH("/:id", h.UpdateExam)
	exams.DELETE("/:id", h.DeleteExam)
	exams.POST("/:id", h.ExamCustom) // :publish, :close, :start, :record
	exams.GET("/:id/questions", h.ExamQuestions)
	exams.GET("/:id/attempt", h.MyAttempt)
	exams.GET("/:id/attempts", h.ListAttempts)

	attempts := offering.Group("/attempts")
	gates.Classroom(attempts, authz.ResourceExam)
	attempts.GET("/:id", h.GetAttempt)
	attempts.POST("/:id", h.AttemptCustom) // :save, :submit, :grade, :review

	attendance := offering.Group("/attendance")
	gates.Classroom(attendance, authz.ResourceAttendance)
	attendance.GET("", h.OfferingAttendance)
	attendance.GET("/summary", h.AttendanceSummaries)
	attendance.GET("/lessons/:id", h.LessonAttendance)
	attendance.POST("/lessons/:id", h.AttendanceCustom) // :initialize, :mark, :excuse
	attendance.PATCH("/records/:id", h.UpdateAttendance)
	attendance.GET("/excuses", h.ListExcuses)
	attendance.POST("/excuses/:id", h.ExcuseCustom) // :review

	grades := offering.Group("/grades")
	gates.Classroom(grades, authz.ResourceGrade)
	grades.GET("/rules", h.GetRules)
	grades.POST("/rules", h.SaveRules)
	grades.GET("", h.ListGrades)
	grades.POST("/finalize", h.FinalizeGrades)
	grades.POST("/definalize", h.DefinalizeGrades)
	grades.PATCH("/:id", h.OverrideGrade)
	grades.GET("/:id/preview", h.PreviewGrade)

	qa := offering.Group("/qa")
	gates.Classroom(qa, authz.ResourceQA)
	qa.GET("", h.ListQAQuestions)
	qa.POST("", h.AskQuestion)
	qa.GET("/:id", h.GetQAQuestion)
	qa.PATCH("/:id", h.UpdateQAQuestion)
	qa.DELETE("/:id", h.DeleteQAQuestion)
	qa.POST("/:id", h.QACustom) // :answer, :reject
	qa.GET("/:id/attachments/:attachmentId", h.DownloadQAAttachment)

	projects := offering.Group("/projects")
	gates.Classroom(projects, authz.ResourceProject)
	projects.GET("", h.ListProjects)
	projects.POST("", h.CreateProject)
	projects.GET("/:id", h.GetProject)
	projects.PATCH("/:id", h.UpdateProject)
	projects.DELETE("/:id", h.DeleteProject)
	projects.POST("/:id", h.ProjectCustom) // :attach, :register, :unregister, :formGroups, :save, :submit, :grade
	projects.GET("/:id/attachments/:name", h.DownloadProjectAttachment)
	projects.DELETE("/:id/attachments/:attachmentId", h.DetachProjectFile)
	projects.GET("/:id/registrations", h.ListRegistrations)
	projects.GET("/:id/groups", h.ListGroups)
	projects.GET("/:id/groups/me", h.MyGroup)
	projects.GET("/:id/submission", h.MyProjectSubmission)
	projects.GET("/:id/submissions", h.ListProjectSubmissions)
	projects.GET("/:id/submissions/:submissionId/files/:name", h.DownloadProjectSubmissionFile)
	projects.GET("/:id/submissions/:submissionId/grades", h.ListProjectGrades)
	projects.GET("/:id/grade", h.MyProjectGrade)

	// Teams: no offering, no gate; leadership and membership are the law.
	teams := rg.Group("/teams")
	teams.GET("", h.MyTeams)
	teams.POST("", h.CreateTeam)
	teams.GET("/:id", h.GetTeam)
	teams.PATCH("/:id", h.RenameTeam)
	teams.DELETE("/:id", h.DeleteTeam)
	teams.POST("/:id/members", h.AddTeamMember)
	teams.DELETE("/:id/members/:userId", h.RemoveTeamMember)
	teams.POST("/:id/leave", h.LeaveTeam)
	teams.POST("/:id/transfer", h.TransferTeamLeadership)

	// The reader's own cross-offering reads, scoped to them by construction.
	rg.GET("/me/classes", h.MyClasses)
	rg.GET("/me/attendance", h.MyCourseAttendance)
}

// respondError is the context's single error→status translation point.
func (h *Handler) respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, classroom.ErrSectionNotFound),
		errors.Is(err, classroom.ErrLessonNotFound),
		errors.Is(err, classroom.ErrAttachmentNotFound),
		errors.Is(err, classroom.ErrScheduleNotFound),
		errors.Is(err, classroom.ErrAssignmentNotFound),
		errors.Is(err, classroom.ErrSubmissionNotFound),
		errors.Is(err, classroom.ErrQuestionNotFound),
		errors.Is(err, classroom.ErrExamNotFound),
		errors.Is(err, classroom.ErrAttemptNotFound),
		errors.Is(err, classroom.ErrAttendanceNotFound),
		errors.Is(err, classroom.ErrExcuseNotFound),
		errors.Is(err, classroom.ErrRulesNotFound),
		errors.Is(err, classroom.ErrAnswerNotFound),
		errors.Is(err, classroom.ErrTeamNotFound),
		errors.Is(err, classroom.ErrMemberNotFound),
		errors.Is(err, classroom.ErrProjectNotFound),
		errors.Is(err, classroom.ErrGroupNotFound),
		errors.Is(err, classroom.ErrCohortGroupNotFound),
		errors.Is(err, classroom.ErrUploadNotFound):
		response.NotFound(c, err.Error())

	case errors.Is(err, classroom.ErrConflict),
		errors.Is(err, classroom.ErrSectionNotEmpty),
		errors.Is(err, classroom.ErrDuplicateName),
		errors.Is(err, classroom.ErrDuplicateSchedule),
		errors.Is(err, classroom.ErrAlreadySubmitted),
		errors.Is(err, classroom.ErrSubmissionGraded),
		errors.Is(err, classroom.ErrExamNotDraft),
		errors.Is(err, classroom.ErrExamNotPublished),
		errors.Is(err, classroom.ErrAttemptSubmitted),
		errors.Is(err, classroom.ErrExcuseExists),
		errors.Is(err, classroom.ErrExcuseReviewed),
		errors.Is(err, classroom.ErrAlreadyFinalized),
		errors.Is(err, classroom.ErrNotFinalized),
		errors.Is(err, classroom.ErrAlreadyMember),
		errors.Is(err, classroom.ErrAlreadyRegistered),
		errors.Is(err, classroom.ErrNotRegistered),
		errors.Is(err, classroom.ErrQuestionNotPending):
		response.Conflict(c, err.Error())

	case errors.Is(err, classroom.ErrNotAuthorized),
		errors.Is(err, classroom.ErrUserMuted),
		errors.Is(err, classroom.ErrNotLeader),
		errors.Is(err, classroom.ErrNotGroupLeader),
		errors.Is(err, classroom.ErrOwnAttendance):
		response.Forbidden(c, err.Error())

	case errors.Is(err, classroom.ErrInvalidInput),
		errors.Is(err, classroom.ErrInvalidPercentage),
		errors.Is(err, classroom.ErrInvalidScore),
		errors.Is(err, classroom.ErrInvalidRules),
		errors.Is(err, classroom.ErrRuleExamNotFound),
		errors.Is(err, classroom.ErrNoContent),
		errors.Is(err, classroom.ErrNoQuestionsInExam),
		errors.Is(err, classroom.ErrTeamSizeInvalid),
		errors.Is(err, classroom.ErrMembersNotEnrolled),
		errors.Is(err, classroom.ErrNoEnrollments):
		response.BadRequest(c, err.Error())

	// Refusals of currently-closed operations: the request was well-formed
	// and permitted, the state disallows it.
	case errors.Is(err, classroom.ErrLessonLocked),
		errors.Is(err, classroom.ErrNotPublished),
		errors.Is(err, classroom.ErrSubmissionsClosed),
		errors.Is(err, classroom.ErrMaxAttemptsReached),
		errors.Is(err, classroom.ErrExamNotAvailable),
		errors.Is(err, classroom.ErrExamNotManual),
		errors.Is(err, classroom.ErrAttendanceNotRequired),
		errors.Is(err, classroom.ErrRegistrationClosed),
		errors.Is(err, classroom.ErrSemesterNotGrading),
		errors.Is(err, classroom.ErrSemesterArchived),
		errors.Is(err, classroom.ErrUngradedWork),
		errors.Is(err, classroom.ErrQuestionRejected),
		errors.Is(err, classroom.ErrTeamFull),
		errors.Is(err, classroom.ErrTeamArchived),
		errors.Is(err, classroom.ErrTeamLocked),
		errors.Is(err, classroom.ErrLeaderCannotLeave),
		errors.Is(err, classroom.ErrNotMember):
		response.Err(c, 422, "UNPROCESSABLE", err.Error())

	default:
		h.log.Error("classroom handler error", zap.Error(err))
		response.InternalError(c)
	}
}

// offeringID is the gated offering; the classroom gate parsed and checked
// it before the handler ran.
func offeringID(c *gin.Context) uuid.UUID {
	return authzhttp.Access(c).OfferingID()
}

// targetID is the ":id" path target, already parsed by the gate.
func targetID(c *gin.Context) uuid.UUID {
	return authzhttp.Access(c).TargetID()
}

// studentView reports whether the caller entered on a student-side seat —
// the flag behind every "students see less" branch.
func studentView(c *gin.Context) bool {
	switch authzhttp.Access(c).Relation() {
	case authz.OfferingRoleStudent, authz.OfferingRoleObserver:
		return true
	case authz.OfferingRoleTeacher, authz.OfferingRoleAssistant, authz.RelationNone:
		return false
	}
	return false
}

// teaching reports whether the caller entered as teaching staff: an
// offering seat with authority, or a staff permission from the policy.
func teaching(c *gin.Context) bool {
	access := authzhttp.Access(c)
	switch access.Relation() {
	case authz.OfferingRoleTeacher, authz.OfferingRoleAssistant:
		return true
	case authz.OfferingRoleStudent, authz.OfferingRoleObserver:
		return false
	case authz.RelationNone:
	}
	return access.Matched() != nil && access.Matched().Type == authz.TypeStaff
}

// requireTeaching aborts with 403 unless the caller is teaching staff;
// used on reads whose route-level action students legitimately hold for
// their own narrower variant.
func requireTeaching(c *gin.Context) bool {
	if !teaching(c) {
		response.Forbidden(c, "permission denied")
		return false
	}
	return true
}

// customAction extracts the ":verb" suffix the gate parsed from the ":id"
// param.
func customAction(c *gin.Context) string {
	return string(authzhttp.Access(c).Action())
}
