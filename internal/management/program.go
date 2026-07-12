package management

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/shared/pagination"
)

// ── Value objects ─────────────────────────────────────────────────────────────

// DegreeType is the degree a program awards. The same closed set is a CHECK
// constraint on programs.degree_type.
type DegreeType string

// Degree types.
const (
	DegreeBachelor DegreeType = "bachelor"
	DegreeMaster   DegreeType = "master"
	DegreePhD      DegreeType = "phd"
)

// ValidDegreeType reports whether d is a known degree type.
func ValidDegreeType(d DegreeType) bool {
	switch d {
	case DegreeBachelor, DegreeMaster, DegreePhD:
		return true
	}
	return false
}

// ── Entities ──────────────────────────────────────────────────────────────────

// Program is a degree program owned by a department. MinAge and MaxAge bound
// admission; nil means unenforced.
type Program struct {
	ID            uuid.UUID  `db:"id"`
	DepartmentID  uuid.UUID  `db:"department_id"`
	NameEN        string     `db:"name_en"`
	NameLocal     *string    `db:"name_local"`
	Code          string     `db:"code"`
	DegreeType    DegreeType `db:"degree_type"`
	DurationYears int        `db:"duration_years"`
	TotalCredits  int        `db:"total_credits"`
	MinAge        *int       `db:"min_age"`
	MaxAge        *int       `db:"max_age"`
	Description   *string    `db:"description"`
	IsActive      bool       `db:"is_active"`
	CreatedAt     time.Time  `db:"created_at"`
	UpdatedAt     time.Time  `db:"updated_at"`
	Version       int64      `db:"version"`
}

// ── Ports ─────────────────────────────────────────────────────────────────────

// ProgramRepository persists programs. GetDepartment lets the service verify
// the parent department exists.
//
// GetProgram returns ErrProgramNotFound. UpdateProgram is an optimistic
// compare-and-swap keyed on version: zero rows → ErrConflict.
type ProgramRepository interface {
	CreateProgram(ctx context.Context, program *Program) error
	GetProgram(ctx context.Context, id uuid.UUID) (*Program, error)
	ListPrograms(ctx context.Context, params pagination.PageParams, filter ProgramFilter) ([]Program, bool, error)
	UpdateProgram(ctx context.Context, program *Program, expectedVersion int64) (int64, error)
	ProgramCodeExists(ctx context.Context, departmentID uuid.UUID, code string, excludeID *uuid.UUID) (bool, error)
	CountProgramsByDepartment(ctx context.Context, departmentID uuid.UUID) (int, error)
	GetDepartment(ctx context.Context, id uuid.UUID) (*Department, error)
}

// ── Service input types ───────────────────────────────────────────────────────

// ProgramFilter narrows program lists; nil fields are ignored.
type ProgramFilter struct {
	DepartmentID *uuid.UUID
	DegreeType   *DegreeType
	IsActive     *bool
}

// ProgramUpdate is a partial edit of a program; nil fields are left
// unchanged.
type ProgramUpdate struct {
	NameEN        *string
	NameLocal     *string
	Code          *string
	DegreeType    *DegreeType
	DurationYears *int
	TotalCredits  *int
	MinAge        *int
	MaxAge        *int
	Description   *string
	IsActive      *bool
}

// ── Service ───────────────────────────────────────────────────────────────────

// ProgramService manages degree programs under the institution's subscription
// limits.
type ProgramService struct {
	repo   ProgramRepository
	limits LimitsProvider
}

// NewProgramService wires a program service.
func NewProgramService(repo ProgramRepository, limits LimitsProvider) *ProgramService {
	return &ProgramService{repo: repo, limits: limits}
}

// Create adds a program to a department, enforcing the per-department program
// limit and code uniqueness within the department.
func (s *ProgramService) Create(ctx context.Context, program *Program) (*Program, error) {
	if !ValidDegreeType(program.DegreeType) {
		return nil, ErrInvalidStatus
	}
	if _, err := s.repo.GetDepartment(ctx, program.DepartmentID); err != nil {
		return nil, err
	}

	limits, err := s.limits.GetLimits(ctx)
	if err != nil {
		return nil, err
	}
	count, err := s.repo.CountProgramsByDepartment(ctx, program.DepartmentID)
	if err != nil {
		return nil, err
	}
	if !CanCreateWithinLimit(count, limits.MaxProgramsPerDepartment) {
		return nil, ErrProgramLimitReached
	}

	exists, err := s.repo.ProgramCodeExists(ctx, program.DepartmentID, program.Code, nil)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrCodeExists
	}

	if err := s.repo.CreateProgram(ctx, program); err != nil {
		return nil, err
	}
	return program, nil
}

// Get fetches one program.
func (s *ProgramService) Get(ctx context.Context, id uuid.UUID) (*Program, error) {
	return s.repo.GetProgram(ctx, id)
}

// List pages through programs matching the filter. A filter on a non-existent
// department is ErrDepartmentNotFound rather than an empty page.
func (s *ProgramService) List(ctx context.Context, params pagination.PageParams, filter ProgramFilter) ([]Program, bool, error) {
	if filter.DepartmentID != nil {
		if _, err := s.repo.GetDepartment(ctx, *filter.DepartmentID); err != nil {
			return nil, false, err
		}
	}
	return s.repo.ListPrograms(ctx, params, filter)
}

// Update applies the patch under optimistic concurrency.
func (s *ProgramService) Update(ctx context.Context, id uuid.UUID, upd ProgramUpdate) (*Program, error) {
	if upd.DegreeType != nil && !ValidDegreeType(*upd.DegreeType) {
		return nil, ErrInvalidStatus
	}
	for attempt := 0; attempt < maxUpdateRetries; attempt++ {
		program, err := s.repo.GetProgram(ctx, id)
		if err != nil {
			return nil, err
		}

		if upd.Code != nil && *upd.Code != program.Code {
			exists, err := s.repo.ProgramCodeExists(ctx, program.DepartmentID, *upd.Code, &id)
			if err != nil {
				return nil, err
			}
			if exists {
				return nil, ErrCodeExists
			}
			program.Code = *upd.Code
		}

		applyProgramPatch(program, upd)

		newVersion, err := s.repo.UpdateProgram(ctx, program, program.Version)
		if errors.Is(err, ErrConflict) {
			continue
		}
		if err != nil {
			return nil, err
		}
		program.Version = newVersion
		return program, nil
	}
	return nil, ErrConflict
}

func applyProgramPatch(program *Program, upd ProgramUpdate) {
	if upd.NameEN != nil {
		program.NameEN = *upd.NameEN
	}
	if upd.NameLocal != nil {
		program.NameLocal = upd.NameLocal
	}
	if upd.Description != nil {
		program.Description = upd.Description
	}
	if upd.DegreeType != nil {
		program.DegreeType = *upd.DegreeType
	}
	if upd.DurationYears != nil {
		program.DurationYears = *upd.DurationYears
	}
	if upd.TotalCredits != nil {
		program.TotalCredits = *upd.TotalCredits
	}
	if upd.MinAge != nil {
		program.MinAge = upd.MinAge
	}
	if upd.MaxAge != nil {
		program.MaxAge = upd.MaxAge
	}
	if upd.IsActive != nil {
		program.IsActive = *upd.IsActive
	}
}
