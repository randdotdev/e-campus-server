package enrollment

import (
	"time"

	"github.com/google/uuid"
)

type Enrollment struct {
	ID             uuid.UUID  `db:"id"`
	OfferingID     uuid.UUID  `db:"offering_id"`
	StudentID      uuid.UUID  `db:"student_id"`
	EnrollmentType string     `db:"enrollment_type"`
	Status         string     `db:"status"`
	EnrolledAt     time.Time  `db:"enrolled_at"`
	CompletedAt    *time.Time `db:"completed_at"`
	FinalGrade     *float64   `db:"final_grade"`
}

const (
	EnrollmentTypeCurriculum = "curriculum"
	EnrollmentTypeRetake     = "retake"
	EnrollmentTypePretake    = "pretake"
	EnrollmentTypeExtra      = "extra"
)

const (
	EnrollmentStatusEnrolled       = "enrolled"
	EnrollmentStatusDropped        = "dropped"
	EnrollmentStatusCompleted      = "completed"
	EnrollmentStatusFailed         = "failed"
	EnrollmentStatusWithdrawnLeave = "withdrawn_leave"
)

type AccessLevel int

const (
	NoAccess AccessLevel = iota
	ViewOnly
	FullAccess
)

func (a AccessLevel) String() string {
	switch a {
	case FullAccess:
		return "full"
	case ViewOnly:
		return "view_only"
	default:
		return "none"
	}
}

type CohortGroup struct {
	ID         uuid.UUID `db:"id"`
	ProgramID  uuid.UUID `db:"program_id"`
	CohortYear int       `db:"cohort_year"`
	Stage      int       `db:"stage"`
	Type       string    `db:"type"`
	Name       string    `db:"name"`
	CreatedAt  time.Time `db:"created_at"`
}

type StudentCohortGroup struct {
	ID            uuid.UUID `db:"id"`
	StudentID     uuid.UUID `db:"student_id"`
	CohortGroupID uuid.UUID `db:"cohort_group_id"`
	AssignedAt    time.Time `db:"assigned_at"`
}

type ProjectGroup struct {
	ID         uuid.UUID `db:"id"`
	OfferingID uuid.UUID `db:"offering_id"`
	Type       string    `db:"type"`
	Name       string    `db:"name"`
	CreatedAt  time.Time `db:"created_at"`
}

type ProjectGroupMember struct {
	ID             uuid.UUID `db:"id"`
	StudentID      uuid.UUID `db:"student_id"`
	ProjectGroupID uuid.UUID `db:"group_id"`
	AssignedAt     time.Time `db:"assigned_at"`
}

const (
	GroupTypeTheory   = "theory"
	GroupTypePractice = "practice"
)

type Request struct {
	ID              uuid.UUID  `db:"id"`
	Type            string     `db:"type"`
	StudentID       uuid.UUID  `db:"student_id"`
	CourseID        uuid.UUID  `db:"course_id"`
	SemesterID      uuid.UUID  `db:"semester_id"`
	Reason          string     `db:"reason"`
	Status          string     `db:"status"`
	ReviewedBy      *uuid.UUID `db:"reviewed_by"`
	ReviewedAt      *time.Time `db:"reviewed_at"`
	RejectionReason *string    `db:"rejection_reason"`
	CreatedAt       time.Time  `db:"created_at"`
}

const (
	TypePretake = "pretake"
	TypeRetake  = "retake"
)

const (
	StatusPending  = "pending"
	StatusApproved = "approved"
	StatusRejected = "rejected"
)

type PrereqStatus struct {
	CourseID        uuid.UUID
	CourseCode      string
	CourseNameEN    string
	CourseNameLocal *string
	Status          string
}

const (
	PrereqNotTaken   = "not_taken"
	PrereqInProgress = "in_progress"
	PrereqFailed     = "failed"
	PrereqPassed     = "passed"
)

type CourseStatus struct {
	CourseID        uuid.UUID
	CourseCode      string
	CourseNameEN    string
	CourseNameLocal *string
	Status          string
	IsNaturalCohort bool
}

const (
	CourseNotTaken   = "not_taken"
	CourseInProgress = "in_progress"
	CourseFailed     = "failed"
	CoursePassed     = "passed"
)

type Warning struct {
	Type         string  `json:"type"`
	Status       string  `json:"status"`
	MessageEN    string  `json:"message_en"`
	MessageLocal *string `json:"message_local,omitempty"`
}
