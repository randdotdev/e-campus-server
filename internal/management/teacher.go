package management

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ── Value objects ─────────────────────────────────────────────────────────────

// TeacherRole is a staff member's authority inside one offering. The same
// closed set is a CHECK constraint on offering_teachers.role.
type TeacherRole string

// Teacher roles.
const (
	RoleTeacher   TeacherRole = "teacher"
	RoleAssistant TeacherRole = "assistant"
	RoleObserver  TeacherRole = "observer"
)

// ValidTeacherRole reports whether r is a known teacher role.
func ValidTeacherRole(r TeacherRole) bool {
	switch r {
	case RoleTeacher, RoleAssistant, RoleObserver:
		return true
	}
	return false
}

// ── Entities ──────────────────────────────────────────────────────────────────

// Teacher assigns a user to an offering with a role.
type Teacher struct {
	ID         uuid.UUID   `db:"id"`
	OfferingID uuid.UUID   `db:"offering_id"`
	UserID     uuid.UUID   `db:"user_id"`
	Role       TeacherRole `db:"role"`
	CreatedAt  time.Time   `db:"created_at"`
}

// ── Derived read models ───────────────────────────────────────────────────────

// TeacherWithUser is the assignment joined with the user's display columns
// (offering_teachers ⋈ users, the published identity columns).
type TeacherWithUser struct {
	Teacher
	UserFullNameEN    string  `db:"user_full_name_en"`
	UserFullNameLocal *string `db:"user_full_name_local"`
	UserEmail         string  `db:"user_email"`
}

// MyTeachingOffering is one row of a staff member's own teaching list
// (offering_teachers ⋈ course_offerings ⋈ courses).
type MyTeachingOffering struct {
	OfferingID      uuid.UUID   `db:"offering_id"`
	Role            TeacherRole `db:"role"`
	CourseID        uuid.UUID   `db:"course_id"`
	CourseCode      string      `db:"course_code"`
	CourseNameEN    string      `db:"course_name_en"`
	CourseNameLocal *string     `db:"course_name_local"`
	CohortYear      int         `db:"cohort_year"`
	Shift           Shift       `db:"shift"`
	IsActive        bool        `db:"is_active"`
	SemesterID      uuid.UUID   `db:"semester_id"`
}

// ── Pure domain rules ─────────────────────────────────────────────────────────

// CanTeacherManage reports whether the role may manage offering content.
func CanTeacherManage(r TeacherRole) bool { return r == RoleTeacher || r == RoleAssistant }

// CanTeacherGrade reports whether the role may grade students.
func CanTeacherGrade(r TeacherRole) bool { return r == RoleTeacher }

// ── Ports ─────────────────────────────────────────────────────────────────────

// TeacherRepository persists teaching assignments.
//
// CreateTeacher returns ErrAlreadyTeacher when the user is already assigned
// to the offering — the unique constraint is the guard. GetTeacher returns
// ErrTeacherNotFound. UpdateTeacherRole returns ErrTeacherNotFound when no
// assignment exists. DeleteTeacher is idempotent.
type TeacherRepository interface {
	CreateTeacher(ctx context.Context, t *Teacher) error
	GetTeacher(ctx context.Context, offeringID, userID uuid.UUID) (*Teacher, error)
	ListTeachers(ctx context.Context, offeringID uuid.UUID) ([]TeacherWithUser, error)
	UpdateTeacherRole(ctx context.Context, offeringID, userID uuid.UUID, role TeacherRole) error
	DeleteTeacher(ctx context.Context, offeringID, userID uuid.UUID) error
	ListMyTeachingOfferings(ctx context.Context, userID uuid.UUID) ([]MyTeachingOffering, error)
	OfferingExists(ctx context.Context, id uuid.UUID) (bool, error)
}

// ── Service ───────────────────────────────────────────────────────────────────

// TeacherService manages the teaching staff of offerings.
type TeacherService struct {
	repo TeacherRepository
}

// NewTeacherService wires a teacher service.
func NewTeacherService(repo TeacherRepository) *TeacherService {
	return &TeacherService{repo: repo}
}

// CreateTeacher assigns a user to an offering. The duplicate guard is the
// unique (offering, user) constraint.
func (s *TeacherService) CreateTeacher(ctx context.Context, offeringID, userID uuid.UUID, role TeacherRole) (*Teacher, error) {
	if !ValidTeacherRole(role) {
		return nil, ErrInvalidStatus
	}
	exists, err := s.repo.OfferingExists(ctx, offeringID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrOfferingNotFound
	}
	teacher := &Teacher{OfferingID: offeringID, UserID: userID, Role: role}
	if err := s.repo.CreateTeacher(ctx, teacher); err != nil {
		return nil, err
	}
	return teacher, nil
}

// GetTeacher fetches a user's assignment in an offering.
func (s *TeacherService) GetTeacher(ctx context.Context, offeringID, userID uuid.UUID) (*Teacher, error) {
	return s.repo.GetTeacher(ctx, offeringID, userID)
}

// ListTeachers returns an offering's teaching staff with display columns.
func (s *TeacherService) ListTeachers(ctx context.Context, offeringID uuid.UUID) ([]TeacherWithUser, error) {
	exists, err := s.repo.OfferingExists(ctx, offeringID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrOfferingNotFound
	}
	return s.repo.ListTeachers(ctx, offeringID)
}

// UpdateTeacherRole changes a user's role in an offering.
func (s *TeacherService) UpdateTeacherRole(ctx context.Context, offeringID, userID uuid.UUID, role TeacherRole) error {
	if !ValidTeacherRole(role) {
		return ErrInvalidStatus
	}
	return s.repo.UpdateTeacherRole(ctx, offeringID, userID, role)
}

// DeleteTeacher removes a user's assignment from an offering.
func (s *TeacherService) DeleteTeacher(ctx context.Context, offeringID, userID uuid.UUID) error {
	return s.repo.DeleteTeacher(ctx, offeringID, userID)
}

// ListMyTeachingOfferings returns the caller's own teaching assignments.
func (s *TeacherService) ListMyTeachingOfferings(ctx context.Context, userID uuid.UUID) ([]MyTeachingOffering, error) {
	return s.repo.ListMyTeachingOfferings(ctx, userID)
}
