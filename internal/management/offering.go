package management

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/shared/pagination"
)

// ── Entities ──────────────────────────────────────────────────────────────────

// Offering is one course taught in one semester to one cohort and shift.
type Offering struct {
	ID         uuid.UUID `db:"id"`
	CourseID   uuid.UUID `db:"course_id"`
	SemesterID uuid.UUID `db:"semester_id"`
	CohortYear int       `db:"cohort_year"`
	Shift      Shift     `db:"shift"`
	IsActive   bool      `db:"is_active"`
	CreatedAt  time.Time `db:"created_at"`
	Version    int64     `db:"version"`
}

// ── Derived read models ───────────────────────────────────────────────────────

// RichOffering is the offering joined with its course's display columns
// (course_offerings ⋈ courses).
type RichOffering struct {
	Offering
	CourseCode      string    `db:"course_code"`
	CourseNameEN    string    `db:"course_name_en"`
	CourseNameLocal *string   `db:"course_name_local"`
	DepartmentID    uuid.UUID `db:"department_id"`
}

// AcademicOfferingInfo is the slim offering projection the semester service
// consumes during offering generation and bulk enrollment.
type AcademicOfferingInfo struct {
	ID       uuid.UUID `db:"id"`
	CourseID uuid.UUID `db:"course_id"`
}

// ── Ports ─────────────────────────────────────────────────────────────────────

// OfferingRepository persists course offerings.
//
// CreateOffering returns ErrDuplicateOffering when the (course, semester,
// cohort, shift) tuple exists — the unique constraint is the guard.
// GetOffering returns ErrOfferingNotFound. UpdateOffering is an optimistic
// compare-and-swap keyed on version. DeleteOffering is idempotent.
type OfferingRepository interface {
	CreateOffering(ctx context.Context, o *Offering) error
	GetOffering(ctx context.Context, id uuid.UUID) (*Offering, error)
	ListOfferings(ctx context.Context, params pagination.PageParams, filter OfferingFilter) ([]Offering, bool, error)
	ListRichOfferings(ctx context.Context, params pagination.PageParams, filter OfferingFilter) ([]RichOffering, bool, error)
	UpdateOffering(ctx context.Context, o *Offering, expectedVersion int64) (int64, error)
	DeleteOffering(ctx context.Context, id uuid.UUID) error
	SemesterExists(ctx context.Context, semesterID uuid.UUID) (bool, error)
	CourseExists(ctx context.Context, courseID uuid.UUID) (bool, error)
}

// ── Service input types ───────────────────────────────────────────────────────

// OfferingFilter narrows offering lists; nil fields are ignored.
type OfferingFilter struct {
	CourseID   *uuid.UUID
	SemesterID *uuid.UUID
	Shift      *Shift
	CohortYear *int
	IsActive   *bool
	CollegeID  *uuid.UUID
	Scope      ScopeFilter
}

// OfferingUpdate is a partial edit of an offering; nil fields are left
// unchanged.
type OfferingUpdate struct {
	IsActive *bool
}

// ── Service ───────────────────────────────────────────────────────────────────

// OfferingService manages course offerings.
type OfferingService struct {
	repo OfferingRepository
}

// NewOfferingService wires an offering service.
func NewOfferingService(repo OfferingRepository) *OfferingService {
	return &OfferingService{repo: repo}
}

// CreateOffering schedules a course for a semester, cohort, and shift.
func (s *OfferingService) CreateOffering(ctx context.Context, offering *Offering) (*Offering, error) {
	if !ValidShift(offering.Shift) {
		return nil, ErrInvalidStatus
	}
	exists, err := s.repo.CourseExists(ctx, offering.CourseID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrCourseNotFound
	}
	exists, err = s.repo.SemesterExists(ctx, offering.SemesterID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrSemesterNotFound
	}
	if err := s.repo.CreateOffering(ctx, offering); err != nil {
		return nil, err
	}
	return offering, nil
}

// GetOffering fetches one offering.
func (s *OfferingService) GetOffering(ctx context.Context, id uuid.UUID) (*Offering, error) {
	return s.repo.GetOffering(ctx, id)
}

// ListOfferings pages through offerings matching the filter.
func (s *OfferingService) ListOfferings(ctx context.Context, params pagination.PageParams, filter OfferingFilter) ([]Offering, bool, error) {
	return s.repo.ListOfferings(ctx, params, filter)
}

// ListRichOfferings pages through offerings with course display columns.
func (s *OfferingService) ListRichOfferings(ctx context.Context, params pagination.PageParams, filter OfferingFilter) ([]RichOffering, bool, error) {
	return s.repo.ListRichOfferings(ctx, params, filter)
}

// UpdateOffering applies the patch under optimistic concurrency.
func (s *OfferingService) UpdateOffering(ctx context.Context, id uuid.UUID, upd OfferingUpdate) (*Offering, error) {
	for attempt := 0; attempt < maxUpdateRetries; attempt++ {
		offering, err := s.repo.GetOffering(ctx, id)
		if err != nil {
			return nil, err
		}
		if upd.IsActive != nil {
			offering.IsActive = *upd.IsActive
		}
		newVersion, err := s.repo.UpdateOffering(ctx, offering, offering.Version)
		if errors.Is(err, ErrConflict) {
			continue
		}
		if err != nil {
			return nil, err
		}
		offering.Version = newVersion
		return offering, nil
	}
	return nil, ErrConflict
}

// DeleteOffering removes an offering.
func (s *OfferingService) DeleteOffering(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteOffering(ctx, id)
}
