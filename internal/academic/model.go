package academic

import (
	"time"

	"github.com/google/uuid"
)

type AcademicYear struct {
	ID        uuid.UUID `db:"id"`
	Year      int       `db:"year"`
	StartDate time.Time `db:"start_date"`
	EndDate   time.Time `db:"end_date"`
	Status    string    `db:"status"`
	CreatedAt time.Time `db:"created_at"`
}

const (
	AcademicYearStatusUpcoming  = "upcoming"
	AcademicYearStatusActive    = "active"
	AcademicYearStatusFinalized = "finalized"
	AcademicYearStatusArchived  = "archived"
)

type Semester struct {
	ID                uuid.UUID  `db:"id"`
	AcademicYearID    uuid.UUID  `db:"academic_year_id"`
	Semester          string     `db:"semester"`
	StartDate         time.Time  `db:"start_date"`
	EndDate           time.Time  `db:"end_date"`
	RegistrationStart *time.Time `db:"registration_start"`
	RegistrationEnd   *time.Time `db:"registration_end"`
	GradeEntryStart   *time.Time `db:"grade_entry_start"`
	GradeEntryEnd     *time.Time `db:"grade_entry_end"`
	PassThreshold     int        `db:"pass_threshold"`
	Status            string     `db:"status"`
	CreatedAt         time.Time  `db:"created_at"`
}

const (
	SemesterTypeFall   = "fall"
	SemesterTypeSpring = "spring"
	SemesterTypeSummer = "summer"
	SemesterTypeAnnual = "annual"
)

const (
	SemesterStatusUpcoming  = "upcoming"
	SemesterStatusActive    = "active"
	SemesterStatusGrading   = "grading"
	SemesterStatusFinalized = "finalized"
	SemesterStatusArchived  = "archived"
)

const (
	ShiftDay     = "day"
	ShiftEvening = "evening"
)

const (
	OfferingStatusCreated = "created"
	OfferingStatusSkipped = "skipped"
	OfferingStatusError   = "error"
)

const (
	StudentStatusActive = "active"
)

type Curriculum struct {
	ID         uuid.UUID `db:"id"`
	ProgramID  uuid.UUID `db:"program_id"`
	CohortYear int       `db:"cohort_year"`
	Stage      int       `db:"stage"`
	Semester   string    `db:"semester"`
	CourseID   uuid.UUID `db:"course_id"`
	IsRequired bool      `db:"is_required"`
	CreatedAt  time.Time `db:"created_at"`
}

type SemesterRequirement struct {
	ID         uuid.UUID `db:"id"`
	ProgramID  uuid.UUID `db:"program_id"`
	CohortYear int       `db:"cohort_year"`
	Stage      int       `db:"stage"`
	Semester   string    `db:"semester"`
	MinCredits int       `db:"min_credits"`
	CreatedBy  uuid.UUID `db:"created_by"`
	CreatedAt  time.Time `db:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"`
}

type BulkEnrollResult struct {
	Enrolled int            `json:"enrolled"`
	Skipped  int            `json:"skipped"`
	Blocked  int            `json:"blocked"`
	Details  *EnrollDetails `json:"details,omitempty"`
}

type EnrollDetails struct {
	Enrolled []EnrollRecord  `json:"enrolled,omitempty"`
	Skipped  []SkipRecord    `json:"skipped,omitempty"`
	Blocked  []BlockedRecord `json:"blocked,omitempty"`
}

type EnrollRecord struct {
	StudentID   uuid.UUID `json:"student_id"`
	StudentName string    `json:"student_name"`
	OfferingID  uuid.UUID `json:"offering_id"`
	CourseCode  string    `json:"course_code"`
	Type        string    `json:"type"`
}

type SkipRecord struct {
	StudentID   uuid.UUID `json:"student_id"`
	StudentName string    `json:"student_name"`
	CourseID    uuid.UUID `json:"course_id"`
	CourseCode  string    `json:"course_code"`
	Reason      string    `json:"reason"`
}

type BlockedRecord struct {
	StudentID            uuid.UUID `json:"student_id"`
	StudentName          string    `json:"student_name"`
	CourseCode           string    `json:"course_code"`
	MissingPrerequisite  string    `json:"missing_prerequisite"`
	MissingCourseID      uuid.UUID `json:"missing_course_id"`
}

type GenerateOfferingsResult struct {
	Created int              `json:"created"`
	Skipped int              `json:"skipped"`
	Details []OfferingRecord `json:"details,omitempty"`
}

type OfferingRecord struct {
	CourseID   uuid.UUID `json:"course_id"`
	CourseCode string    `json:"course_code"`
	CohortYear int       `json:"cohort_year"`
	Shift      string    `json:"shift"`
	Status     string    `json:"status"`
}

type EndSemesterResult struct {
	Processed int `json:"processed"`
	Promoted  int `json:"promoted"`
	Repeated  int `json:"repeated"`
	Unchanged int `json:"unchanged"`
	Errors    int `json:"errors"`
}
