package management

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/shared/pagination"
)

// ── Value objects ─────────────────────────────────────────────────────────────

// StudentStatus is the student's standing in the university. The same closed
// set is a CHECK constraint on students.status.
type StudentStatus string

// Student statuses.
const (
	StudentActive    StudentStatus = "active"
	StudentGraduated StudentStatus = "graduated"
	StudentWithdrawn StudentStatus = "withdrawn"
	StudentSuspended StudentStatus = "suspended"
	StudentOnLeave   StudentStatus = "on_leave"
)

// ValidStudentStatus reports whether s is a known student status.
func ValidStudentStatus(s StudentStatus) bool {
	switch s {
	case StudentActive, StudentGraduated, StudentWithdrawn, StudentSuspended, StudentOnLeave:
		return true
	}
	return false
}

// Shift is the teaching shift a student attends and an offering is taught in.
// The same closed set is a CHECK constraint on students.shift,
// applications.shift, and course_offerings.shift.
type Shift string

// Shifts.
const (
	ShiftDay     Shift = "day"
	ShiftEvening Shift = "evening"
)

// ValidShift reports whether s is a known shift.
func ValidShift(s Shift) bool { return s == ShiftDay || s == ShiftEvening }

// Tuition is the payment basis of a student's seat. The same closed set is a
// CHECK constraint on students.tuition and applications.tuition.
type Tuition string

// Tuition kinds.
const (
	TuitionFree Tuition = "free"
	TuitionPaid Tuition = "paid"
)

// ValidTuition reports whether t is a known tuition kind.
func ValidTuition(t Tuition) bool { return t == TuitionFree || t == TuitionPaid }

// CohortChangeReason records why a student moved between cohorts.
type CohortChangeReason string

// Cohort change reasons.
const (
	CohortChangeFailed      CohortChangeReason = "failed"
	CohortChangeTransferred CohortChangeReason = "transferred"
	CohortChangeReturned    CohortChangeReason = "returned"
)

// ── Entities ──────────────────────────────────────────────────────────────────

// Student is a person admitted into a program, keyed by the account
// (users.id) — one account has at most one student record, so no surrogate
// id exists. CurrentCohortYear tracks the intake class the student
// progresses with (it diverges from AdmissionYear when a year is repeated);
// CurrentYear is the study stage within the program.
type Student struct {
	UserID            uuid.UUID     `db:"user_id"`
	ProgramID         uuid.UUID     `db:"program_id"`
	AdmissionYear     int           `db:"admission_year"`
	CurrentCohortYear int           `db:"current_cohort_year"`
	CurrentYear       int           `db:"current_year"`
	Shift             Shift         `db:"shift"`
	Tuition           Tuition       `db:"tuition"`
	Status            StudentStatus `db:"status"`
	EnrolledAt        time.Time     `db:"enrolled_at"`
	CreatedAt         time.Time     `db:"created_at"`
	Version           int64         `db:"version"`
}

// CohortHistory is one recorded cohort/stage move of a student. StudentID
// is the account id (users.id), like every student reference.
type CohortHistory struct {
	ID             uuid.UUID          `db:"id"`
	StudentID      uuid.UUID          `db:"student_id"`
	FromCohortYear int                `db:"from_cohort_year"`
	ToCohortYear   int                `db:"to_cohort_year"`
	FromYear       int                `db:"from_year"`
	ToYear         int                `db:"to_year"`
	Reason         CohortChangeReason `db:"reason"`
	Notes          *string            `db:"notes"`
	ChangedAt      time.Time          `db:"changed_at"`
}

// ── Derived read models ───────────────────────────────────────────────────────

// StudentSummary is the student row joined with the user's display name
// (students ⋈ users, the published identity columns).
type StudentSummary struct {
	Student
	NameEN    string  `db:"name_en"`
	NameLocal *string `db:"name_local"`
}

// CohortYearSummary is the per-cohort head count of a program
// (GROUP BY over students).
type CohortYearSummary struct {
	CohortYear   int `db:"cohort_year"`
	StudentCount int `db:"student_count"`
}

// StudentScope locates a student's place in the university structure
// (students ⋈ programs ⋈ departments), the projection peer contexts consume
// to answer "may this user see content scoped to X".
type StudentScope struct {
	UserID       uuid.UUID     `db:"user_id"`
	ProgramID    uuid.UUID     `db:"program_id"`
	DepartmentID uuid.UUID     `db:"department_id"`
	CollegeID    uuid.UUID     `db:"college_id"`
	Status       StudentStatus `db:"status"`
}

// AcademicStudentInfo is the slim student projection the semester service
// consumes for offering generation, bulk enrollment, and progression
// (students ⋈ users for the display name).
type AcademicStudentInfo struct {
	UserID            uuid.UUID
	ProgramID         uuid.UUID
	CurrentYear       int
	CurrentCohortYear int
	Status            StudentStatus
	Name              string
	Shift             Shift
}

// ── Ports ─────────────────────────────────────────────────────────────────────

// StudentRepository persists students and their cohort history. Students
// key on the account id (users.id) everywhere.
//
// CreateStudent returns ErrDuplicateStudent when the user already has a
// student record (students PK = user_id) and ErrProgramNotFound when the
// program reference is broken. GetStudent returns ErrStudentNotFound.
// UpdateStudent is an optimistic compare-and-swap keyed on version: zero
// rows → ErrConflict (or ErrStudentNotFound when the row is gone).
type StudentRepository interface {
	CreateStudent(ctx context.Context, s *Student) error
	GetStudent(ctx context.Context, userID uuid.UUID) (*StudentSummary, error)
	ListStudents(ctx context.Context, params pagination.PageParams, filter StudentFilter) ([]StudentSummary, bool, error)
	UpdateStudent(ctx context.Context, s *Student, expectedVersion int64) (int64, error)
	ListCohortYears(ctx context.Context, programID uuid.UUID) ([]CohortYearSummary, error)
	ListCohortHistory(ctx context.Context, studentID uuid.UUID) ([]CohortHistory, error)
	GetTranscriptData(ctx context.Context, studentID uuid.UUID) (*TranscriptData, error)
	// GetStudentScope returns nil (no error) when the user has no student
	// record.
	GetStudentScope(ctx context.Context, userID uuid.UUID) (*StudentScope, error)
}

// StudentProgramProvider is what the student service needs to know about
// programs. ProgramExists never errors on absence; GetProgramTotalCredits
// returns ErrProgramNotFound.
type StudentProgramProvider interface {
	ProgramExists(ctx context.Context, id uuid.UUID) (bool, error)
	GetProgramTotalCredits(ctx context.Context, id uuid.UUID) (int, error)
}

// ── Service input types ───────────────────────────────────────────────────────

// StudentFilter narrows student lists; nil fields are ignored.
type StudentFilter struct {
	ProgramID     *uuid.UUID
	CohortYear    *int
	Stage         *int
	Status        *StudentStatus
	Shift         *Shift
	CohortGroupID *uuid.UUID
	Query         *string
	Scope         ScopeFilter
}

// StudentUpdate is a partial edit of a student record; nil fields are left
// unchanged.
type StudentUpdate struct {
	CurrentYear       *int
	CurrentCohortYear *int
	Shift             *Shift
	Tuition           *Tuition
}

// ── Service ───────────────────────────────────────────────────────────────────

// StudentService manages student records: admission, edits, status changes,
// cohort history, and the transcript.
type StudentService struct {
	repo    StudentRepository
	program StudentProgramProvider
}

// NewStudentService wires a student service.
func NewStudentService(repo StudentRepository, program StudentProgramProvider) *StudentService {
	return &StudentService{repo: repo, program: program}
}

// CreateStudent admits a user into a program. The student starts active, in
// stage one, with the cohort year equal to the admission year. The duplicate
// guard is the unique index on students.user_id, not the Go-side check.
func (s *StudentService) CreateStudent(ctx context.Context, userID, programID uuid.UUID, admissionYear int, shift Shift, tuition Tuition) (*StudentSummary, error) {
	if !ValidShift(shift) || !ValidTuition(tuition) {
		return nil, ErrInvalidStatus
	}
	exists, err := s.program.ProgramExists(ctx, programID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrProgramNotFound
	}

	student := &Student{
		UserID:            userID,
		ProgramID:         programID,
		AdmissionYear:     admissionYear,
		CurrentCohortYear: admissionYear,
		CurrentYear:       1,
		Shift:             shift,
		Tuition:           tuition,
		Status:            StudentActive,
	}
	if err := s.repo.CreateStudent(ctx, student); err != nil {
		return nil, err
	}
	return s.repo.GetStudent(ctx, student.UserID)
}

// GetStudent fetches one student with display fields by account id.
func (s *StudentService) GetStudent(ctx context.Context, userID uuid.UUID) (*StudentSummary, error) {
	return s.repo.GetStudent(ctx, userID)
}

// ListStudents pages through students matching the filter.
func (s *StudentService) ListStudents(ctx context.Context, params pagination.PageParams, filter StudentFilter) ([]StudentSummary, bool, error) {
	return s.repo.ListStudents(ctx, params, filter)
}

// ListCohortYears returns the cohort head counts of a program.
func (s *StudentService) ListCohortYears(ctx context.Context, programID uuid.UUID) ([]CohortYearSummary, error) {
	return s.repo.ListCohortYears(ctx, programID)
}

// ListCohortHistory returns a student's cohort moves, newest first.
func (s *StudentService) ListCohortHistory(ctx context.Context, studentID uuid.UUID) ([]CohortHistory, error) {
	return s.repo.ListCohortHistory(ctx, studentID)
}

// UpdateStudent applies the patch under optimistic concurrency: each attempt
// re-reads the row, re-applies the patch, and compare-and-swaps on version,
// so concurrent edits to different fields merge instead of clobbering.
func (s *StudentService) UpdateStudent(ctx context.Context, id uuid.UUID, upd StudentUpdate) (*StudentSummary, error) {
	if upd.Shift != nil && !ValidShift(*upd.Shift) {
		return nil, ErrInvalidStatus
	}
	if upd.Tuition != nil && !ValidTuition(*upd.Tuition) {
		return nil, ErrInvalidStatus
	}
	for attempt := 0; attempt < maxUpdateRetries; attempt++ {
		student, err := s.repo.GetStudent(ctx, id)
		if err != nil {
			return nil, err
		}
		if upd.CurrentYear != nil {
			student.CurrentYear = *upd.CurrentYear
		}
		if upd.CurrentCohortYear != nil {
			student.CurrentCohortYear = *upd.CurrentCohortYear
		}
		if upd.Shift != nil {
			student.Shift = *upd.Shift
		}
		if upd.Tuition != nil {
			student.Tuition = *upd.Tuition
		}
		newVersion, err := s.repo.UpdateStudent(ctx, &student.Student, student.Version)
		if errors.Is(err, ErrConflict) {
			continue
		}
		if err != nil {
			return nil, err
		}
		student.Version = newVersion
		return student, nil
	}
	return nil, ErrConflict
}

// UpdateStudentStatus sets the student's standing. Any transition is legal by
// policy (admins correct records freely), so the write is a plain version CAS
// rather than a guarded state transition.
func (s *StudentService) UpdateStudentStatus(ctx context.Context, id uuid.UUID, status StudentStatus) (*StudentSummary, error) {
	if !ValidStudentStatus(status) {
		return nil, ErrInvalidStatus
	}
	for attempt := 0; attempt < maxUpdateRetries; attempt++ {
		student, err := s.repo.GetStudent(ctx, id)
		if err != nil {
			return nil, err
		}
		student.Status = status
		newVersion, err := s.repo.UpdateStudent(ctx, &student.Student, student.Version)
		if errors.Is(err, ErrConflict) {
			continue
		}
		if err != nil {
			return nil, err
		}
		student.Version = newVersion
		return student, nil
	}
	return nil, ErrConflict
}

// GetStudentScope locates the user's student record in the university
// structure, or nil when the user is not a student.
func (s *StudentService) GetStudentScope(ctx context.Context, userID uuid.UUID) (*StudentScope, error) {
	return s.repo.GetStudentScope(ctx, userID)
}

// GetTranscript assembles the student's full transcript.
func (s *StudentService) GetTranscript(ctx context.Context, studentID uuid.UUID) (*Transcript, error) {
	student, err := s.repo.GetStudent(ctx, studentID)
	if err != nil {
		return nil, err
	}
	data, err := s.repo.GetTranscriptData(ctx, studentID)
	if err != nil {
		return nil, err
	}
	totalCredits, err := s.program.GetProgramTotalCredits(ctx, student.ProgramID)
	if err != nil {
		return nil, err
	}
	return BuildTranscript(data, student, totalCredits), nil
}
