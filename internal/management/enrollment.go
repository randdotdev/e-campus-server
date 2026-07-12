package management

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/shared/pagination"
)

// ── Value objects ─────────────────────────────────────────────────────────────

// EnrollmentType is how a student ended up in an offering. The same closed set
// is a CHECK constraint on course_enrollments.enrollment_type.
type EnrollmentType string

// Enrollment types.
const (
	EnrollmentCurriculum EnrollmentType = "curriculum"
	EnrollmentRetake     EnrollmentType = "retake"
	EnrollmentPretake    EnrollmentType = "pretake"
	EnrollmentExtra      EnrollmentType = "extra"
)

// ValidEnrollmentType reports whether t is a known enrollment type.
func ValidEnrollmentType(t EnrollmentType) bool {
	switch t {
	case EnrollmentCurriculum, EnrollmentRetake, EnrollmentPretake, EnrollmentExtra:
		return true
	}
	return false
}

// EnrollmentStatus is the state of one course attempt. The same closed set is
// a CHECK constraint on course_enrollments.status.
type EnrollmentStatus string

// Enrollment statuses.
const (
	EnrollmentEnrolled       EnrollmentStatus = "enrolled"
	EnrollmentDropped        EnrollmentStatus = "dropped"
	EnrollmentCompleted      EnrollmentStatus = "completed"
	EnrollmentFailed         EnrollmentStatus = "failed"
	EnrollmentWithdrawnLeave EnrollmentStatus = "withdrawn_leave"
)

// ValidEnrollmentStatus reports whether s is a known enrollment status.
func ValidEnrollmentStatus(s EnrollmentStatus) bool {
	switch s {
	case EnrollmentEnrolled, EnrollmentDropped, EnrollmentCompleted, EnrollmentFailed, EnrollmentWithdrawnLeave:
		return true
	}
	return false
}

// AccessLevel is a student's access to an offering's classroom content.
type AccessLevel int

// Access levels, weakest to strongest.
const (
	NoAccess AccessLevel = iota
	ViewOnly
	FullAccess
)

// String renders the access level for API responses.
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

// ── Entities ──────────────────────────────────────────────────────────────────

// Enrollment is one student's attempt at one offering. StudentID references
// the user's ID (course_enrollments.student_id → users.id), a historical
// schema decision every consumer of this noun must respect.
type Enrollment struct {
	ID             uuid.UUID        `db:"id"`
	OfferingID     uuid.UUID        `db:"offering_id"`
	StudentID      uuid.UUID        `db:"student_id"`
	EnrollmentType EnrollmentType   `db:"enrollment_type"`
	Status         EnrollmentStatus `db:"status"`
	EnrolledAt     time.Time        `db:"enrolled_at"`
	CompletedAt    *time.Time       `db:"completed_at"`
	FinalGrade     *float64         `db:"final_grade"`
}

// ── Derived read models ───────────────────────────────────────────────────────

// FinalGradeRow is one line of an offering's grade sheet
// (course_enrollments ⋈ users) — read by classroom through its
// GradeWriter port.
type FinalGradeRow struct {
	StudentID   uuid.UUID `db:"student_id"`
	StudentName string    `db:"student_name"`
	FinalGrade  *float64  `db:"final_grade"`
	Status      string    `db:"status"`
}

// EnrollmentWithStudent is the enrollment joined with the student's display
// columns (course_enrollments ⋈ users, the published identity columns).
type EnrollmentWithStudent struct {
	Enrollment
	StudentFullNameEN    string  `db:"student_full_name_en"`
	StudentFullNameLocal *string `db:"student_full_name_local"`
	StudentEmail         string  `db:"student_email"`
}

// MyEnrollment is one row of a student's own course list (course_enrollments
// ⋈ course_offerings ⋈ courses ⋈ semesters).
type MyEnrollment struct {
	ID             uuid.UUID        `db:"id"`
	OfferingID     uuid.UUID        `db:"offering_id"`
	CourseName     string           `db:"course_name"`
	CourseCode     string           `db:"course_code"`
	SemesterName   SemesterType     `db:"semester_name"`
	EnrollmentType EnrollmentType   `db:"enrollment_type"`
	Status         EnrollmentStatus `db:"status"`
	EnrolledAt     time.Time        `db:"enrolled_at"`
	CompletedAt    *time.Time       `db:"completed_at"`
	FinalGrade     *float64         `db:"final_grade"`
}

// OfferingInfo is the slim offering projection the enrollment service needs
// for sibling-enrollment checks.
type OfferingInfo struct {
	ID         uuid.UUID `db:"id"`
	CourseID   uuid.UUID `db:"course_id"`
	SemesterID uuid.UUID `db:"semester_id"`
	CohortYear int       `db:"cohort_year"`
	Shift      Shift     `db:"shift"`
}

// CourseInfo is the slim course projection the enrollment service needs to
// resolve a course's department and code.
type CourseInfo struct {
	ID           uuid.UUID `db:"id"`
	DepartmentID uuid.UUID `db:"department_id"`
	Code         string    `db:"code"`
}

// ── Pure domain rules ─────────────────────────────────────────────────────────

// ResolveAccessLevel derives classroom access from the student's own
// enrollment and any enrollment in a sibling offering of the same course.
func ResolveAccessLevel(isEnrolled, hasSiblingEnrollment bool) AccessLevel {
	if isEnrolled {
		return FullAccess
	}
	if hasSiblingEnrollment {
		return ViewOnly
	}
	return NoAccess
}

// ── Ports ─────────────────────────────────────────────────────────────────────

// EnrollmentOfferingReader is what the enrollment service needs from the
// offering catalogue.
type EnrollmentOfferingReader interface {
	OfferingExists(ctx context.Context, id uuid.UUID) (bool, error)
	GetOfferingInfo(ctx context.Context, id uuid.UUID) (*OfferingInfo, error)
	GetOfferingsInfoByCourseCodeAndCohort(ctx context.Context, departmentID uuid.UUID, code string, cohortYear int, shift Shift) ([]OfferingInfo, error)
}

// EnrollmentCourseReader is what the enrollment service needs from the course
// catalogue.
type EnrollmentCourseReader interface {
	GetCourseInfo(ctx context.Context, id uuid.UUID) (*CourseInfo, error)
}

// EnrollmentRepository persists course enrollments.
//
// CreateEnrollment returns ErrAlreadyEnrolled when the (offering, student)
// pair exists — enforced by the unique constraint, not a prior read.
// GetEnrollment returns nil (no error) when the pair does not exist.
// DropEnrollment moves an enrollment to dropped only from enrolled, in one
// guarded UPDATE; a miss is ErrEnrollmentNotFound.
type EnrollmentRepository interface {
	CreateEnrollment(ctx context.Context, e *Enrollment) error
	GetEnrollment(ctx context.Context, offeringID, studentID uuid.UUID) (*Enrollment, error)
	ListEnrollments(ctx context.Context, params pagination.PageParams, filter EnrollmentFilter) ([]EnrollmentWithStudent, bool, error)
	IsEnrolled(ctx context.Context, offeringID, studentID uuid.UUID) (bool, error)
	GetEnrolledStudentIDs(ctx context.Context, offeringID uuid.UUID) ([]uuid.UUID, error)
	GetMyEnrollments(ctx context.Context, studentID uuid.UUID, status *EnrollmentStatus) ([]MyEnrollment, error)
	DropEnrollment(ctx context.Context, enrollmentID uuid.UUID) error

	// The final-grade surface classroom writes through (§11): landing a
	// grade guards on an active enrollment inside the statement.
	SetFinalGrade(ctx context.Context, offeringID, studentID uuid.UUID, grade float64, status string) error
	ClearFinalGrades(ctx context.Context, offeringID uuid.UUID) error
	IsOfferingFinalized(ctx context.Context, offeringID uuid.UUID) (bool, error)
	GetFinalGrades(ctx context.Context, offeringID uuid.UUID) ([]FinalGradeRow, error)
}

// ── Service input types ───────────────────────────────────────────────────────

// EnrollmentFilter narrows enrollment lists; nil fields are ignored.
type EnrollmentFilter struct {
	OfferingID     *uuid.UUID
	EnrollmentType *EnrollmentType
	Status         *EnrollmentStatus
	Query          string
}

// ── Service ───────────────────────────────────────────────────────────────────

// EnrollmentService manages course enrollments and classroom access levels.
type EnrollmentService struct {
	repo     EnrollmentRepository
	offering EnrollmentOfferingReader
	course   EnrollmentCourseReader
	log      *slog.Logger
}

// NewEnrollmentService wires an enrollment service.
func NewEnrollmentService(repo EnrollmentRepository, offering EnrollmentOfferingReader, course EnrollmentCourseReader, log *slog.Logger) *EnrollmentService {
	return &EnrollmentService{repo: repo, offering: offering, course: course, log: log}
}

// EnrollStudent enrolls a student in an offering. The duplicate guard is the
// unique (offering, student) constraint; a race surfaces as
// ErrAlreadyEnrolled.
func (s *EnrollmentService) EnrollStudent(ctx context.Context, offeringID, studentID uuid.UUID, enrollmentType EnrollmentType) (*Enrollment, error) {
	if enrollmentType == "" {
		enrollmentType = EnrollmentCurriculum
	}
	if !ValidEnrollmentType(enrollmentType) {
		return nil, ErrInvalidRequestType
	}
	exists, err := s.offering.OfferingExists(ctx, offeringID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrOfferingNotFound
	}

	e := &Enrollment{
		OfferingID:     offeringID,
		StudentID:      studentID,
		EnrollmentType: enrollmentType,
		Status:         EnrollmentEnrolled,
	}
	if err := s.repo.CreateEnrollment(ctx, e); err != nil {
		return nil, err
	}
	return e, nil
}

// ListEnrollments pages through enrollments matching the filter.
func (s *EnrollmentService) ListEnrollments(ctx context.Context, params pagination.PageParams, filter EnrollmentFilter) ([]EnrollmentWithStudent, bool, error) {
	return s.repo.ListEnrollments(ctx, params, filter)
}

// GetAccessLevel resolves the student's classroom access to an offering,
// checking sibling offerings of the same course for view-only access.
func (s *EnrollmentService) GetAccessLevel(ctx context.Context, offeringID, studentID uuid.UUID) (AccessLevel, error) {
	isEnrolled, err := s.repo.IsEnrolled(ctx, offeringID, studentID)
	if err != nil {
		return NoAccess, err
	}
	if isEnrolled {
		return FullAccess, nil
	}

	offering, err := s.offering.GetOfferingInfo(ctx, offeringID)
	if err != nil {
		return NoAccess, err
	}
	course, err := s.course.GetCourseInfo(ctx, offering.CourseID)
	if err != nil {
		return NoAccess, err
	}
	siblings, err := s.offering.GetOfferingsInfoByCourseCodeAndCohort(ctx, course.DepartmentID, course.Code, offering.CohortYear, offering.Shift)
	if err != nil {
		return NoAccess, err
	}
	for _, sib := range siblings {
		if sib.ID == offeringID {
			continue
		}
		sibEnrolled, err := s.repo.IsEnrolled(ctx, sib.ID, studentID)
		if err != nil {
			return NoAccess, err
		}
		if sibEnrolled {
			return ResolveAccessLevel(false, true), nil
		}
	}
	return NoAccess, nil
}

// IsEnrolled reports whether the student is actively enrolled in the offering.
func (s *EnrollmentService) IsEnrolled(ctx context.Context, offeringID, studentID uuid.UUID) (bool, error) {
	return s.repo.IsEnrolled(ctx, offeringID, studentID)
}

// GetEnrolledStudentIDs returns the actively enrolled student IDs of an
// offering.
func (s *EnrollmentService) GetEnrolledStudentIDs(ctx context.Context, offeringID uuid.UUID) ([]uuid.UUID, error) {
	return s.repo.GetEnrolledStudentIDs(ctx, offeringID)
}

// SetFinalGrade lands one computed grade on an enrollment; classroom's
// grading calls it and the error propagates (§17).
func (s *EnrollmentService) SetFinalGrade(ctx context.Context, offeringID, studentID uuid.UUID, grade float64, status string) error {
	return s.repo.SetFinalGrade(ctx, offeringID, studentID, grade, status)
}

// ClearFinalGrades reopens an offering's grading.
func (s *EnrollmentService) ClearFinalGrades(ctx context.Context, offeringID uuid.UUID) error {
	return s.repo.ClearFinalGrades(ctx, offeringID)
}

// IsOfferingFinalized reports whether every enrollment carries an outcome.
func (s *EnrollmentService) IsOfferingFinalized(ctx context.Context, offeringID uuid.UUID) (bool, error) {
	return s.repo.IsOfferingFinalized(ctx, offeringID)
}

// FinalGrades is the offering's grade sheet.
func (s *EnrollmentService) FinalGrades(ctx context.Context, offeringID uuid.UUID) ([]FinalGradeRow, error) {
	return s.repo.GetFinalGrades(ctx, offeringID)
}

// GetMyEnrollments returns the student's own course list, optionally filtered
// by status.
func (s *EnrollmentService) GetMyEnrollments(ctx context.Context, studentID uuid.UUID, status *EnrollmentStatus) ([]MyEnrollment, error) {
	return s.repo.GetMyEnrollments(ctx, studentID, status)
}

// DropEnrollment drops the student's active enrollment in an offering.
func (s *EnrollmentService) DropEnrollment(ctx context.Context, offeringID, studentID uuid.UUID) error {
	enrollment, err := s.repo.GetEnrollment(ctx, offeringID, studentID)
	if err != nil {
		return err
	}
	if enrollment == nil {
		return ErrNotEnrolled
	}
	if err := s.repo.DropEnrollment(ctx, enrollment.ID); err != nil {
		return err
	}
	return nil
}
