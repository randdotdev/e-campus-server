package classroom

import "errors"

var (
	ErrSectionNotFound    = errors.New("classroom: section not found")
	ErrLessonNotFound     = errors.New("classroom: lesson not found")
	ErrAttachmentNotFound = errors.New("classroom: attachment not found")
	ErrScheduleNotFound   = errors.New("classroom: schedule not found")
	ErrAssignmentNotFound = errors.New("classroom: assignment not found")
	ErrSubmissionNotFound = errors.New("classroom: submission not found")
	ErrQuestionNotFound   = errors.New("classroom: question not found")
	ErrExamNotFound       = errors.New("classroom: exam not found")
	ErrAttemptNotFound    = errors.New("classroom: attempt not found")
	ErrAttendanceNotFound = errors.New("classroom: attendance not found")
	ErrExcuseNotFound     = errors.New("classroom: excuse not found")
	ErrRulesNotFound      = errors.New("classroom: grading rules not found")
	ErrAnswerNotFound     = errors.New("classroom: answer not found")
	ErrTeamNotFound       = errors.New("classroom: team not found")
	ErrMemberNotFound     = errors.New("classroom: member not found")
	ErrProjectNotFound    = errors.New("classroom: project not found")
	ErrGroupNotFound      = errors.New("classroom: project group not found")

	// ErrConflict is a lost version race; the client refetches and retries.
	ErrConflict = errors.New("classroom: conflict, stale version")

	// ErrNotAuthorized covers refusals the offering gate cannot see: author-
	// only edits, leader-only team moves, owner-only reads.
	ErrNotAuthorized = errors.New("classroom: not authorized")

	ErrInvalidInput = errors.New("classroom: invalid input")

	// content
	ErrSectionNotEmpty     = errors.New("classroom: section still has lessons")
	ErrDuplicateName       = errors.New("classroom: name already used here")
	ErrDuplicateSchedule   = errors.New("classroom: group already scheduled for this lesson")
	ErrCohortGroupNotFound = errors.New("classroom: cohort group not found")
	ErrUploadNotFound      = errors.New("classroom: upload not found")
	ErrLessonLocked        = errors.New("classroom: lesson is not unlocked yet")

	// assignment & project submissions
	ErrNotPublished      = errors.New("classroom: not published")
	ErrSubmissionsClosed = errors.New("classroom: deadline passed")
	ErrAlreadySubmitted  = errors.New("classroom: already submitted")
	ErrSubmissionGraded  = errors.New("classroom: submission already graded")
	ErrNoContent         = errors.New("classroom: submission has no content")
	ErrInvalidScore      = errors.New("classroom: score out of range")

	// exam
	ErrExamNotDraft       = errors.New("classroom: exam is no longer a draft")
	ErrExamNotPublished   = errors.New("classroom: exam is not published")
	ErrExamNotAvailable   = errors.New("classroom: exam is not open")
	ErrExamNotManual      = errors.New("classroom: exam is not manual mode")
	ErrNoQuestionsInExam  = errors.New("classroom: exam has no questions")
	ErrMaxAttemptsReached = errors.New("classroom: attempt limit reached")
	ErrAttemptSubmitted   = errors.New("classroom: attempt already submitted")

	// attendance
	ErrAttendanceNotRequired = errors.New("classroom: lesson does not take attendance")
	ErrInvalidPercentage     = errors.New("classroom: invalid attendance percentage")
	ErrExcuseExists          = errors.New("classroom: excuse already requested")
	ErrExcuseReviewed        = errors.New("classroom: excuse already reviewed")
	ErrOwnAttendance         = errors.New("classroom: cannot review own excuse")

	// grading
	ErrInvalidRules       = errors.New("classroom: rule weights must sum to 100")
	ErrRuleExamNotFound   = errors.New("classroom: rule references a foreign exam")
	ErrSemesterNotGrading = errors.New("classroom: semester is not in grading")
	ErrSemesterArchived   = errors.New("classroom: semester is archived")
	ErrAlreadyFinalized   = errors.New("classroom: grades already finalized")
	ErrNotFinalized       = errors.New("classroom: grades not finalized")
	ErrUngradedWork       = errors.New("classroom: ungraded work remains")
	ErrNoEnrollments      = errors.New("classroom: offering has no enrollments")

	// qa
	ErrQuestionRejected   = errors.New("classroom: question was rejected")
	ErrQuestionNotPending = errors.New("classroom: question is not pending")
	ErrUserMuted          = errors.New("classroom: user is muted")

	// team
	ErrNotLeader         = errors.New("classroom: not the team leader")
	ErrLeaderCannotLeave = errors.New("classroom: leader cannot leave; transfer first")
	ErrAlreadyMember     = errors.New("classroom: already a member")
	ErrNotMember         = errors.New("classroom: not a member")
	ErrTeamFull          = errors.New("classroom: team is full")
	ErrTeamArchived      = errors.New("classroom: team is archived")
	// ErrTeamLocked refuses membership changes once the team has carried a
	// submission — history must keep pointing at the people who made it.
	ErrTeamLocked = errors.New("classroom: team has submissions and is locked")
	// ErrNotClassmate: teams bind to a program and cohort; only its
	// students join.
	ErrNotClassmate = errors.New("classroom: not a student of the team's program and cohort")
	ErrNotStudent   = errors.New("classroom: no student record")

	// project
	ErrRegistrationClosed = errors.New("classroom: registration closed")
	ErrAlreadyRegistered  = errors.New("classroom: team already registered")
	ErrNotRegistered      = errors.New("classroom: team not registered")
	ErrTeamSizeInvalid    = errors.New("classroom: team size outside project bounds")
	ErrMembersNotEnrolled = errors.New("classroom: not every member is enrolled")
	ErrNotGroupLeader     = errors.New("classroom: not the group leader")
)
