package classroom

import (
	"context"

	"github.com/google/uuid"
)

// EnrollmentReader answers who is enrolled, by account (users.id) — the
// key enrollments themselves use. The offering gate already proves the
// caller's own seat; these exist for the people who are not the caller:
// rosters to initialize attendance and compute grades, and other students'
// seats when a team registers.
type EnrollmentReader interface {
	EnrolledUserIDs(ctx context.Context, offeringID uuid.UUID) ([]uuid.UUID, error)
	AllEnrolled(ctx context.Context, offeringID uuid.UUID, userIDs []uuid.UUID) (bool, error)
}

// OfferingReader resolves offering facts that live in management's tables.
type OfferingReader interface {
	CourseCodeByOffering(ctx context.Context, offeringID uuid.UUID) (string, error)
	// SemesterStatus is the offering's semester lifecycle state; grading
	// uses it as a UX pre-check — the write itself is guarded in SQL.
	SemesterStatus(ctx context.Context, offeringID uuid.UUID) (string, error)
	PassThreshold(ctx context.Context, offeringID uuid.UUID) (int, error)
}

// GradeWriter is the one sanctioned write into management: finalized course
// grades land on the enrollment rows management owns. Wired to management's
// enrollment service in main.go; its error always propagates (a grade that
// silently failed to land is a lie).
type GradeWriter interface {
	SetEnrollmentGrade(ctx context.Context, offeringID, studentID uuid.UUID, grade float64, status string) error
	ClearEnrollmentGrades(ctx context.Context, offeringID uuid.UUID) error
	OfferingFinalized(ctx context.Context, offeringID uuid.UUID) (bool, error)
	StudentGrades(ctx context.Context, offeringID uuid.UUID) ([]StudentGrade, error)
}

// StudentReader resolves a person's student facts (management's students
// table): the program and cohort a team binds to at creation.
type StudentReader interface {
	StudentProgramCohort(ctx context.Context, userID uuid.UUID) (programID uuid.UUID, cohortYear int, err error)
}

// CohortGroupReader validates schedule targets and resolves the caller's
// own groups so their sessions can be marked in lesson views.
type CohortGroupReader interface {
	CohortGroupExists(ctx context.Context, id uuid.UUID) (bool, error)
	StudentCohortGroupIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
}
