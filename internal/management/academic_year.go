package management

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ── Value objects ─────────────────────────────────────────────────────────────

// AcademicYearStatus is the academic year's lifecycle state. The same closed
// set is a CHECK constraint on academic_years.status.
type AcademicYearStatus string

// Academic year statuses.
const (
	AcademicYearUpcoming  AcademicYearStatus = "upcoming"
	AcademicYearActive    AcademicYearStatus = "active"
	AcademicYearFinalized AcademicYearStatus = "finalized"
	AcademicYearArchived  AcademicYearStatus = "archived"
)

// ValidAcademicYearStatus reports whether s is a known academic year status.
func ValidAcademicYearStatus(s AcademicYearStatus) bool {
	switch s {
	case AcademicYearUpcoming, AcademicYearActive, AcademicYearFinalized, AcademicYearArchived:
		return true
	}
	return false
}

// ── Entities ──────────────────────────────────────────────────────────────────

// AcademicYear is one academic year (Year is its starting calendar year).
type AcademicYear struct {
	ID        uuid.UUID          `db:"id"`
	Year      int                `db:"year"`
	StartDate time.Time          `db:"start_date"`
	EndDate   time.Time          `db:"end_date"`
	Status    AcademicYearStatus `db:"status"`
	CreatedAt time.Time          `db:"created_at"`
	Version   int64              `db:"version"`
}

// ── Ports ─────────────────────────────────────────────────────────────────────

// AcademicYearRepository persists academic years.
//
// CreateAcademicYear returns ErrDuplicateYear when the year exists.
// GetAcademicYear returns ErrAcademicYearNotFound. UpdateAcademicYear is an
// optimistic compare-and-swap keyed on version: zero rows → ErrConflict.
type AcademicYearRepository interface {
	CreateAcademicYear(ctx context.Context, ay *AcademicYear) error
	GetAcademicYear(ctx context.Context, id uuid.UUID) (*AcademicYear, error)
	ListAcademicYears(ctx context.Context) ([]AcademicYear, error)
	UpdateAcademicYear(ctx context.Context, ay *AcademicYear, expectedVersion int64) (int64, error)
	AcademicYearExists(ctx context.Context, year int) (bool, error)
}

// ── Service input types ───────────────────────────────────────────────────────

// AcademicYearUpdate is a partial edit of an academic year; nil fields are
// left unchanged.
type AcademicYearUpdate struct {
	StartDate *time.Time
	EndDate   *time.Time
	Status    *AcademicYearStatus
}

// ── Service ───────────────────────────────────────────────────────────────────

// AcademicYearService manages the academic calendar's years.
type AcademicYearService struct {
	repo AcademicYearRepository
}

// NewAcademicYearService wires an academic year service.
func NewAcademicYearService(repo AcademicYearRepository) *AcademicYearService {
	return &AcademicYearService{repo: repo}
}

// Create adds an academic year; it starts upcoming.
func (s *AcademicYearService) Create(ctx context.Context, year int, startDate, endDate time.Time) (*AcademicYear, error) {
	exists, err := s.repo.AcademicYearExists(ctx, year)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrDuplicateYear
	}
	ay := &AcademicYear{
		Year:      year,
		StartDate: startDate,
		EndDate:   endDate,
		Status:    AcademicYearUpcoming,
	}
	if err := s.repo.CreateAcademicYear(ctx, ay); err != nil {
		return nil, err
	}
	return ay, nil
}

// Get fetches one academic year.
func (s *AcademicYearService) Get(ctx context.Context, id uuid.UUID) (*AcademicYear, error) {
	return s.repo.GetAcademicYear(ctx, id)
}

// List returns all academic years.
func (s *AcademicYearService) List(ctx context.Context) ([]AcademicYear, error) {
	return s.repo.ListAcademicYears(ctx)
}

// Update applies the patch under optimistic concurrency: each attempt
// re-reads the current row and compare-and-swaps on version, so concurrent
// edits to different fields merge instead of clobbering one another.
func (s *AcademicYearService) Update(ctx context.Context, id uuid.UUID, upd AcademicYearUpdate) (*AcademicYear, error) {
	if upd.Status != nil && !ValidAcademicYearStatus(*upd.Status) {
		return nil, ErrInvalidStatus
	}
	for attempt := 0; attempt < maxUpdateRetries; attempt++ {
		ay, err := s.repo.GetAcademicYear(ctx, id)
		if err != nil {
			return nil, err
		}
		if upd.StartDate != nil {
			ay.StartDate = *upd.StartDate
		}
		if upd.EndDate != nil {
			ay.EndDate = *upd.EndDate
		}
		if upd.Status != nil {
			ay.Status = *upd.Status
		}
		newVersion, err := s.repo.UpdateAcademicYear(ctx, ay, ay.Version)
		if errors.Is(err, ErrConflict) {
			continue
		}
		if err != nil {
			return nil, err
		}
		ay.Version = newVersion
		return ay, nil
	}
	return nil, ErrConflict
}
