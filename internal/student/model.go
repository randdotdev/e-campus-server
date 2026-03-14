package student

import (
	"time"

	"github.com/google/uuid"
)

type Student struct {
	ID               uuid.UUID `db:"id"`
	UserID           uuid.UUID `db:"user_id"`
	ProgramID        uuid.UUID `db:"program_id"`
	AdmissionYear    int       `db:"admission_year"`
	CurrentCohortYear int      `db:"current_cohort_year"`
	CurrentYear      int       `db:"current_year"`
	Shift            string    `db:"shift"`
	Tuition          string    `db:"tuition"`
	Status           string    `db:"status"`
	EnrolledAt       time.Time `db:"enrolled_at"`
	CreatedAt        time.Time `db:"created_at"`
}

const (
	StatusActive    = "active"
	StatusGraduated = "graduated"
	StatusWithdrawn = "withdrawn"
	StatusSuspended = "suspended"
	StatusOnLeave   = "on_leave"
)

const (
	ShiftDay     = "day"
	ShiftEvening = "evening"
)

const (
	TuitionFree = "free"
	TuitionPaid = "paid"
)

type Leave struct {
	ID             uuid.UUID  `db:"id"`
	StudentID      uuid.UUID  `db:"student_id"`
	Type           string     `db:"type"`
	AcademicYearID *uuid.UUID `db:"academic_year_id"`
	Reason         string     `db:"reason"`
	StartDate      *time.Time `db:"start_date"`
	EndDate        *time.Time `db:"end_date"`
	ApprovedBy     *uuid.UUID `db:"approved_by"`
	ApprovedAt     *time.Time `db:"approved_at"`
	Notes          *string    `db:"notes"`
	CreatedAt      time.Time  `db:"created_at"`
}

const (
	LeaveTypeShort    = "short"
	LeaveTypeSemester = "semester"
	LeaveTypeYear     = "year"
)

type LeaveSemester struct {
	LeaveID    uuid.UUID `db:"leave_id"`
	SemesterID uuid.UUID `db:"semester_id"`
}

type CohortHistory struct {
	ID            uuid.UUID `db:"id"`
	StudentID     uuid.UUID `db:"student_id"`
	FromCohortYear int      `db:"from_cohort_year"`
	ToCohortYear  int       `db:"to_cohort_year"`
	FromYear      int       `db:"from_year"`
	ToYear        int       `db:"to_year"`
	Reason        string    `db:"reason"`
	Notes         *string   `db:"notes"`
	ChangedAt     time.Time `db:"changed_at"`
}

const (
	CohortChangeReasonFailed     = "failed"
	CohortChangeReasonTransferred = "transferred"
	CohortChangeReasonReturned   = "returned"
)

type TranscriptEntry struct {
	CourseCode   string   `json:"course_code"`
	CourseName   string   `json:"course_name"`
	Credits      int      `json:"credits"`
	Grade        *float64 `json:"grade"`
	Status       string   `json:"status"`
}

type TranscriptSemester struct {
	AcademicYear    string            `json:"academic_year"`
	Semester        string            `json:"semester"`
	Courses         []TranscriptEntry `json:"courses"`
	SemesterCredits int               `json:"semester_credits"`
	SemesterGPA     float64           `json:"semester_gpa"`
}

type Transcript struct {
	Student   TranscriptStudent    `json:"student"`
	Semesters []TranscriptSemester `json:"semesters"`
	Totals    TranscriptTotals     `json:"totals"`
}

type TranscriptStudent struct {
	Name          string `json:"name"`
	Program       string `json:"program"`
	AdmissionYear int    `json:"admission_year"`
	Status        string `json:"status"`
}

type TranscriptTotals struct {
	CreditsEarned   int     `json:"credits_earned"`
	CreditsRequired int     `json:"credits_required"`
	CumulativeGPA   float64 `json:"cumulative_gpa"`
	ProgressPercent float64 `json:"progress_percent"`
}

type StudentFilters struct {
	ProgramID  *uuid.UUID
	CohortYear *int
	Stage      *int
	Status     *string
	Shift      *string
	Query      *string
}
