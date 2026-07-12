package management

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/shared/pagination"
)

// ── Entities ──────────────────────────────────────────────────────────────────

// Department is a subdivision of a college.
type Department struct {
	ID          uuid.UUID     `db:"id"`
	CollegeID   uuid.UUID     `db:"college_id"`
	NameEN      string        `db:"name_en"`
	NameLocal   *string       `db:"name_local"`
	Code        string        `db:"code"`
	Description LocalizedText `db:"description"`
	IsActive    bool          `db:"is_active"`
	CreatedAt   time.Time     `db:"created_at"`
	UpdatedAt   time.Time     `db:"updated_at"`
	Version     int64         `db:"version"`

	About   LocalizedText `db:"about"`
	Founded *int          `db:"founded"`
	Phone   *string       `db:"phone"`
	Email   *string       `db:"email"`
	LogoURL *string       `db:"logo_url"`
}

// ── Ports ─────────────────────────────────────────────────────────────────────

// DepartmentRepository persists departments. GetCollege lets the service
// verify the parent college exists before a create.
//
// GetDepartment returns ErrDepartmentNotFound. UpdateDepartment is an
// optimistic compare-and-swap keyed on version: zero rows → ErrConflict.
type DepartmentRepository interface {
	CreateDepartment(ctx context.Context, dept *Department) error
	GetDepartment(ctx context.Context, id uuid.UUID) (*Department, error)
	ListDepartments(ctx context.Context, params pagination.PageParams, filter DepartmentFilter) ([]Department, bool, error)
	UpdateDepartment(ctx context.Context, dept *Department, expectedVersion int64) (int64, error)
	DepartmentCodeExists(ctx context.Context, collegeID uuid.UUID, code string, excludeID *uuid.UUID) (bool, error)
	CountDepartmentsByCollege(ctx context.Context, collegeID uuid.UUID) (int, error)
	GetCollege(ctx context.Context, id uuid.UUID) (*College, error)
}

// ── Service input types ───────────────────────────────────────────────────────

// DepartmentFilter narrows department lists; nil fields are ignored.
type DepartmentFilter struct {
	CollegeID *uuid.UUID
	IsActive  *bool
}

// DepartmentUpdate is a partial edit of a department; nil fields are left
// unchanged.
type DepartmentUpdate struct {
	NameEN      *string
	NameLocal   *string
	Code        *string
	Description LocalizedText
	IsActive    *bool
	About       LocalizedText
	Founded     *int
	Phone       *string
	Email       *string
	LogoURL     *string
}

// ── Service ───────────────────────────────────────────────────────────────────

// DepartmentService manages departments under the institution's subscription
// limits.
type DepartmentService struct {
	repo   DepartmentRepository
	limits LimitsProvider
}

// NewDepartmentService wires a department service.
func NewDepartmentService(repo DepartmentRepository, limits LimitsProvider) *DepartmentService {
	return &DepartmentService{repo: repo, limits: limits}
}

// Create adds a department to a college, enforcing the per-college department
// limit and code uniqueness within the college.
func (s *DepartmentService) Create(ctx context.Context, dept *Department) (*Department, error) {
	if _, err := s.repo.GetCollege(ctx, dept.CollegeID); err != nil {
		return nil, err
	}

	limits, err := s.limits.GetLimits(ctx)
	if err != nil {
		return nil, err
	}
	count, err := s.repo.CountDepartmentsByCollege(ctx, dept.CollegeID)
	if err != nil {
		return nil, err
	}
	if !CanCreateWithinLimit(count, limits.MaxDepartmentsPerCollege) {
		return nil, ErrDepartmentLimitReached
	}

	exists, err := s.repo.DepartmentCodeExists(ctx, dept.CollegeID, dept.Code, nil)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrCodeExists
	}

	if err := s.repo.CreateDepartment(ctx, dept); err != nil {
		return nil, err
	}
	return dept, nil
}

// Get fetches one department.
func (s *DepartmentService) Get(ctx context.Context, id uuid.UUID) (*Department, error) {
	return s.repo.GetDepartment(ctx, id)
}

// List pages through departments matching the filter. A filter on a
// non-existent college is ErrCollegeNotFound rather than an empty page.
func (s *DepartmentService) List(ctx context.Context, params pagination.PageParams, filter DepartmentFilter) ([]Department, bool, error) {
	if filter.CollegeID != nil {
		if _, err := s.repo.GetCollege(ctx, *filter.CollegeID); err != nil {
			return nil, false, err
		}
	}
	return s.repo.ListDepartments(ctx, params, filter)
}

// Update applies the patch under optimistic concurrency.
func (s *DepartmentService) Update(ctx context.Context, id uuid.UUID, upd DepartmentUpdate) (*Department, error) {
	for attempt := 0; attempt < maxUpdateRetries; attempt++ {
		dept, err := s.repo.GetDepartment(ctx, id)
		if err != nil {
			return nil, err
		}

		if upd.Code != nil && *upd.Code != dept.Code {
			exists, err := s.repo.DepartmentCodeExists(ctx, dept.CollegeID, *upd.Code, &id)
			if err != nil {
				return nil, err
			}
			if exists {
				return nil, ErrCodeExists
			}
			dept.Code = *upd.Code
		}

		applyDepartmentPatch(dept, upd)

		newVersion, err := s.repo.UpdateDepartment(ctx, dept, dept.Version)
		if errors.Is(err, ErrConflict) {
			continue
		}
		if err != nil {
			return nil, err
		}
		dept.Version = newVersion
		return dept, nil
	}
	return nil, ErrConflict
}

func applyDepartmentPatch(dept *Department, upd DepartmentUpdate) {
	if upd.NameEN != nil {
		dept.NameEN = *upd.NameEN
	}
	if upd.NameLocal != nil {
		dept.NameLocal = upd.NameLocal
	}
	if upd.Description != nil {
		dept.Description = upd.Description
	}
	if upd.IsActive != nil {
		dept.IsActive = *upd.IsActive
	}
	if upd.About != nil {
		dept.About = upd.About
	}
	if upd.Founded != nil {
		dept.Founded = upd.Founded
	}
	if upd.Phone != nil {
		dept.Phone = upd.Phone
	}
	if upd.Email != nil {
		dept.Email = upd.Email
	}
	if upd.LogoURL != nil {
		dept.LogoURL = upd.LogoURL
	}
}
