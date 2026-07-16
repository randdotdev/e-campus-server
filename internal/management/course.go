package management

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/shared/pagination"
)

// ── Entities ──────────────────────────────────────────────────────────────────

// Course is a catalogue entry owned by a department. Courses sharing a code
// within a department are siblings (multi-part courses); GroupOrder orders
// them. Requires points at the prerequisite course, if any.
type Course struct {
	ID               uuid.UUID  `db:"id"`
	DepartmentID     uuid.UUID  `db:"department_id"`
	Code             string     `db:"code"`
	NameEN           string     `db:"name_en"`
	NameLocal        *string    `db:"name_local"`
	SubtitleEN       *string    `db:"subtitle_en"`
	SubtitleLocal    *string    `db:"subtitle_local"`
	GroupOrder       int        `db:"group_order"`
	Requires         *uuid.UUID `db:"requires"`
	Credits          int        `db:"credits"`
	DescriptionEN    *string    `db:"description_en"`
	DescriptionLocal *string    `db:"description_local"`
	IsActive         bool       `db:"is_active"`
	CreatedAt        time.Time  `db:"created_at"`
	UpdatedAt        time.Time  `db:"updated_at"`
	DeletedAt        *time.Time `db:"deleted_at"`
	Version          int64      `db:"version"`
}

// ── Ports ─────────────────────────────────────────────────────────────────────

// CourseRepository persists catalogue courses.
//
// CreateCourse returns ErrDuplicateCode when (department, code, group order)
// already exists. GetCourse returns ErrCourseNotFound. UpdateCourse is an
// optimistic compare-and-swap keyed on version: zero rows → ErrConflict (or
// ErrCourseNotFound when the row is gone). DeleteCourse is idempotent.
type CourseRepository interface {
	CreateCourse(ctx context.Context, c *Course) error
	GetCourse(ctx context.Context, id uuid.UUID) (*Course, error)
	ListCourses(ctx context.Context, params pagination.PageParams, filter CourseFilter) ([]Course, bool, error)
	UpdateCourse(ctx context.Context, c *Course, expectedVersion int64) (int64, error)
	DeleteCourse(ctx context.Context, id uuid.UUID) error
	GetCoursesByCode(ctx context.Context, departmentID uuid.UUID, code string) ([]Course, error)
	CourseCodeExists(ctx context.Context, departmentID uuid.UUID, code string, groupOrder int, excludeID *uuid.UUID) (bool, error)
}

// ── Service input types ───────────────────────────────────────────────────────

// CourseFilter narrows course lists; zero-valued fields are ignored.
type CourseFilter struct {
	DepartmentID *uuid.UUID
	IsActive     *bool
	HasRequires  *bool
	Query        string
}

// CourseUpdate is a partial edit of a course; nil fields are left unchanged.
type CourseUpdate struct {
	NameEN           *string
	NameLocal        *string
	SubtitleEN       *string
	SubtitleLocal    *string
	DescriptionEN    *string
	DescriptionLocal *string
	IsActive         *bool
	Credits          *int
}

// ── Service ───────────────────────────────────────────────────────────────────

// CourseService manages the course catalogue.
type CourseService struct {
	repo CourseRepository
}

// NewCourseService wires a course service.
func NewCourseService(repo CourseRepository) *CourseService {
	return &CourseService{repo: repo}
}

// CreateCourse adds a course to the catalogue. A zero GroupOrder defaults to
// one. The duplicate-code pre-check gives a friendly error; the unique
// constraint is the guard.
func (s *CourseService) CreateCourse(ctx context.Context, course *Course) (*Course, error) {
	if course.GroupOrder == 0 {
		course.GroupOrder = 1
	}
	exists, err := s.repo.CourseCodeExists(ctx, course.DepartmentID, course.Code, course.GroupOrder, nil)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrDuplicateCode
	}
	if err := s.repo.CreateCourse(ctx, course); err != nil {
		return nil, err
	}
	return course, nil
}

// GetCourse fetches one course.
func (s *CourseService) GetCourse(ctx context.Context, id uuid.UUID) (*Course, error) {
	return s.repo.GetCourse(ctx, id)
}

// ListCourses pages through courses matching the filter.
func (s *CourseService) ListCourses(ctx context.Context, params pagination.PageParams, filter CourseFilter) ([]Course, bool, error) {
	return s.repo.ListCourses(ctx, params, filter)
}

// UpdateCourse applies the patch under optimistic concurrency: each attempt
// re-reads the row, re-applies the patch, and compare-and-swaps on version.
func (s *CourseService) UpdateCourse(ctx context.Context, id uuid.UUID, upd CourseUpdate) (*Course, error) {
	for attempt := 0; attempt < maxUpdateRetries; attempt++ {
		course, err := s.repo.GetCourse(ctx, id)
		if err != nil {
			return nil, err
		}
		if upd.NameEN != nil {
			course.NameEN = *upd.NameEN
		}
		if upd.NameLocal != nil {
			course.NameLocal = upd.NameLocal
		}
		if upd.SubtitleEN != nil {
			course.SubtitleEN = upd.SubtitleEN
		}
		if upd.SubtitleLocal != nil {
			course.SubtitleLocal = upd.SubtitleLocal
		}
		if upd.DescriptionEN != nil {
			course.DescriptionEN = upd.DescriptionEN
		}
		if upd.DescriptionLocal != nil {
			course.DescriptionLocal = upd.DescriptionLocal
		}
		if upd.Credits != nil {
			course.Credits = *upd.Credits
		}
		if upd.IsActive != nil {
			course.IsActive = *upd.IsActive
		}
		newVersion, err := s.repo.UpdateCourse(ctx, course, course.Version)
		if errors.Is(err, ErrConflict) {
			continue
		}
		if err != nil {
			return nil, err
		}
		course.Version = newVersion
		return course, nil
	}
	return nil, ErrConflict
}

// DeleteCourse removes a course from the catalogue.
func (s *CourseService) DeleteCourse(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteCourse(ctx, id)
}

// GetSiblingCourses returns all courses sharing the given course's code
// within its department.
func (s *CourseService) GetSiblingCourses(ctx context.Context, courseID uuid.UUID) ([]Course, error) {
	course, err := s.repo.GetCourse(ctx, courseID)
	if err != nil {
		return nil, err
	}
	return s.repo.GetCoursesByCode(ctx, course.DepartmentID, course.Code)
}
