package management

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ── Entities ──────────────────────────────────────────────────────────────────

// Curriculum is one course's place in a program's study plan: which cohort,
// stage, and semester it is taught in. Rows are immutable — the plan changes
// by adding and deleting entries.
type Curriculum struct {
	ID         uuid.UUID    `db:"id"`
	ProgramID  uuid.UUID    `db:"program_id"`
	CohortYear int          `db:"cohort_year"`
	Stage      int          `db:"stage"`
	Semester   SemesterType `db:"semester"`
	CourseID   uuid.UUID    `db:"course_id"`
	IsRequired bool         `db:"is_required"`
	CreatedAt  time.Time    `db:"created_at"`
}

// SemesterRequirement is the minimum credits a student must earn in one
// (program, cohort, stage, semester) cell to pass it.
type SemesterRequirement struct {
	ID         uuid.UUID    `db:"id"`
	ProgramID  uuid.UUID    `db:"program_id"`
	CohortYear int          `db:"cohort_year"`
	Stage      int          `db:"stage"`
	Semester   SemesterType `db:"semester"`
	MinCredits int          `db:"min_credits"`
	CreatedBy  uuid.UUID    `db:"created_by"`
	CreatedAt  time.Time    `db:"created_at"`
	UpdatedAt  time.Time    `db:"updated_at"`
}

// ── Derived read models ───────────────────────────────────────────────────────

// CurriculumItem is a curriculum row with its course's display columns
// (curriculum ⋈ courses).
type CurriculumItem struct {
	ID              uuid.UUID    `db:"id"`
	ProgramID       uuid.UUID    `db:"program_id"`
	CohortYear      int          `db:"cohort_year"`
	Stage           int          `db:"stage"`
	Semester        SemesterType `db:"semester"`
	IsRequired      bool         `db:"is_required"`
	CreatedAt       time.Time    `db:"created_at"`
	CourseID        uuid.UUID    `db:"course_id"`
	CourseCode      string       `db:"course_code"`
	CourseNameEN    string       `db:"course_name_en"`
	CourseNameLocal *string      `db:"course_name_local"`
	CourseCredits   int          `db:"course_credits"`
}

// ── Ports ─────────────────────────────────────────────────────────────────────

// CurriculumRepository persists curriculum entries.
//
// CreateCurriculum returns ErrDuplicateCurriculum when the entry exists — the
// unique constraint is the guard. GetCurriculumByID returns
// ErrCurriculumNotFound. DeleteCurriculum is idempotent.
type CurriculumRepository interface {
	CreateCurriculum(ctx context.Context, c *Curriculum) error
	GetCurriculumByID(ctx context.Context, id uuid.UUID) (*Curriculum, error)
	GetCurriculum(ctx context.Context, programID uuid.UUID, cohortYear, stage int, semester SemesterType) ([]Curriculum, error)
	ListCurriculum(ctx context.Context, programID uuid.UUID, cohortYear int) ([]Curriculum, error)
	ListCurriculumItems(ctx context.Context, programID uuid.UUID, cohortYear int) ([]CurriculumItem, error)
	DeleteCurriculum(ctx context.Context, id uuid.UUID) error
}

// RequirementRepository persists semester requirements. SetRequirement is an
// upsert keyed on (program, cohort, stage, semester).
type RequirementRepository interface {
	SetRequirement(ctx context.Context, r *SemesterRequirement) error
	GetRequirement(ctx context.Context, programID uuid.UUID, cohortYear, stage int, semester SemesterType) (*SemesterRequirement, error)
	ListRequirements(ctx context.Context, programID uuid.UUID, cohortYear int) ([]SemesterRequirement, error)
}

// ── Service ───────────────────────────────────────────────────────────────────

// CurriculumService manages curriculum entries and semester requirements.
// They are co-located because requirements annotate the same (program,
// cohort, stage, semester) key that curriculum entries are grouped by.
type CurriculumService struct {
	curriculum   CurriculumRepository
	requirements RequirementRepository
}

// NewCurriculumService wires a curriculum service.
func NewCurriculumService(curriculum CurriculumRepository, requirements RequirementRepository) *CurriculumService {
	return &CurriculumService{curriculum: curriculum, requirements: requirements}
}

// CreateCurriculum adds a course to a program's study plan. The duplicate
// guard is the unique constraint; a race surfaces as ErrDuplicateCurriculum.
func (s *CurriculumService) CreateCurriculum(ctx context.Context, c *Curriculum) (*Curriculum, error) {
	if !ValidSemesterType(c.Semester) {
		return nil, ErrInvalidStatus
	}
	if err := s.curriculum.CreateCurriculum(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

// GetCurriculumByID fetches one curriculum entry.
func (s *CurriculumService) GetCurriculumByID(ctx context.Context, id uuid.UUID) (*Curriculum, error) {
	return s.curriculum.GetCurriculumByID(ctx, id)
}

// GetCurriculum returns the plan of one (program, cohort, stage, semester)
// cell.
func (s *CurriculumService) GetCurriculum(ctx context.Context, programID uuid.UUID, cohortYear, stage int, semester SemesterType) ([]Curriculum, error) {
	return s.curriculum.GetCurriculum(ctx, programID, cohortYear, stage, semester)
}

// ListCurriculum returns a program cohort's whole plan.
func (s *CurriculumService) ListCurriculum(ctx context.Context, programID uuid.UUID, cohortYear int) ([]Curriculum, error) {
	return s.curriculum.ListCurriculum(ctx, programID, cohortYear)
}

// ListCurriculumItems returns a program cohort's plan with course display
// columns.
func (s *CurriculumService) ListCurriculumItems(ctx context.Context, programID uuid.UUID, cohortYear int) ([]CurriculumItem, error) {
	return s.curriculum.ListCurriculumItems(ctx, programID, cohortYear)
}

// DeleteCurriculum removes a course from a study plan.
func (s *CurriculumService) DeleteCurriculum(ctx context.Context, id uuid.UUID) error {
	return s.curriculum.DeleteCurriculum(ctx, id)
}

// SetRequirement upserts the minimum-credit requirement of one plan cell.
func (s *CurriculumService) SetRequirement(ctx context.Context, r *SemesterRequirement) (*SemesterRequirement, error) {
	if !ValidSemesterType(r.Semester) {
		return nil, ErrInvalidStatus
	}
	if err := s.requirements.SetRequirement(ctx, r); err != nil {
		return nil, err
	}
	return r, nil
}

// GetRequirement fetches the requirement of one plan cell, or
// ErrRequirementNotFound.
func (s *CurriculumService) GetRequirement(ctx context.Context, programID uuid.UUID, cohortYear, stage int, semester SemesterType) (*SemesterRequirement, error) {
	return s.requirements.GetRequirement(ctx, programID, cohortYear, stage, semester)
}

// ListRequirements returns a program cohort's requirements.
func (s *CurriculumService) ListRequirements(ctx context.Context, programID uuid.UUID, cohortYear int) ([]SemesterRequirement, error) {
	return s.requirements.ListRequirements(ctx, programID, cohortYear)
}
