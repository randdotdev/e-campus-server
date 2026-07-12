package management

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ── Value objects ─────────────────────────────────────────────────────────────

// CohortGroupType distinguishes theory and practice cohort groups. The same
// closed set is a CHECK constraint on cohort_groups.type.
type CohortGroupType string

// Cohort group types.
const (
	CohortGroupTheory   CohortGroupType = "theory"
	CohortGroupPractice CohortGroupType = "practice"
)

// ValidCohortGroupType reports whether t is a known cohort group type.
func ValidCohortGroupType(t CohortGroupType) bool {
	return t == CohortGroupTheory || t == CohortGroupPractice
}

// ── Entities ──────────────────────────────────────────────────────────────────

// CohortGroup is a standing subdivision of a program cohort (e.g. "Group A")
// used for scheduling; a student belongs to at most one theory and one
// practice group per cohort. Project teams for group work are a different
// concept and belong to the classroom context.
type CohortGroup struct {
	ID         uuid.UUID       `db:"id"`
	ProgramID  uuid.UUID       `db:"program_id"`
	CohortYear int             `db:"cohort_year"`
	Stage      int             `db:"stage"`
	Type       CohortGroupType `db:"type"`
	Name       string          `db:"name"`
	CreatedAt  time.Time       `db:"created_at"`
}

// StudentCohortGroup is a student's membership in a cohort group.
type StudentCohortGroup struct {
	ID            uuid.UUID `db:"id"`
	StudentID     uuid.UUID `db:"student_id"`
	CohortGroupID uuid.UUID `db:"cohort_group_id"`
	AssignedAt    time.Time `db:"assigned_at"`
}

// ── Derived read models ───────────────────────────────────────────────────────

// CohortGroupWithCount is a cohort group with its member head count
// (cohort_groups with a correlated count over student_cohort_groups).
type CohortGroupWithCount struct {
	CohortGroup
	MemberCount int `db:"member_count"`
}

// ── Pure domain rules ─────────────────────────────────────────────────────────

// PickCohortGroups chooses the least-populated theory and practice group from
// a candidate list; a nil result means no group of that type exists. This is
// the balancing rule ReassignCohortGroups enforces under lock.
func PickCohortGroups(groups []CohortGroupWithCount) (theory, practice *CohortGroupWithCount) {
	for i := range groups {
		g := &groups[i]
		if g.Type == CohortGroupTheory && (theory == nil || g.MemberCount < theory.MemberCount) {
			theory = g
		}
		if g.Type == CohortGroupPractice && (practice == nil || g.MemberCount < practice.MemberCount) {
			practice = g
		}
	}
	return theory, practice
}

// ── Ports ─────────────────────────────────────────────────────────────────────

// CohortGroupRepository persists cohort groups and their memberships.
//
// CreateCohortGroup returns ErrDuplicateCohortGroup when the (program,
// cohort, stage, type, name) key exists — the unique constraint is the guard.
// GetCohortGroup returns nil (no error) when the group does not exist.
// AssignToCohortGroup is idempotent: assigning an existing member is a no-op
// (unique constraint absorbs the race). ReassignCohortGroups atomically
// clears the student's memberships and re-assigns them to the
// least-populated theory and practice groups of the target cohort, locking
// the candidate groups for the duration so concurrent reassignments cannot
// over-fill a group.
type CohortGroupRepository interface {
	CreateCohortGroup(ctx context.Context, g *CohortGroup) error
	GetCohortGroup(ctx context.Context, id uuid.UUID) (*CohortGroup, error)
	ListCohortGroups(ctx context.Context, programID uuid.UUID, cohortYear, stage int) ([]CohortGroup, error)
	ListCohortGroupsWithCounts(ctx context.Context, programID uuid.UUID, cohortYear, stage int) ([]CohortGroupWithCount, error)
	DeleteCohortGroup(ctx context.Context, id uuid.UUID) error
	CohortGroupExists(ctx context.Context, id uuid.UUID) (bool, error)
	AssignToCohortGroup(ctx context.Context, m *StudentCohortGroup) error
	DeleteCohortGroupMember(ctx context.Context, studentID, groupID uuid.UUID) error
	StudentCohortGroupIDs(ctx context.Context, studentID uuid.UUID) ([]uuid.UUID, error)
	ReassignCohortGroups(ctx context.Context, studentID, programID uuid.UUID, cohortYear, stage int) error
}

// ── Service ───────────────────────────────────────────────────────────────────

// CohortGroupService manages the standing theory/practice groups of program
// cohorts.
type CohortGroupService struct {
	repo CohortGroupRepository
}

// NewCohortGroupService wires a cohort group service.
func NewCohortGroupService(repo CohortGroupRepository) *CohortGroupService {
	return &CohortGroupService{repo: repo}
}

// CreateCohortGroup creates a standing group for a program cohort.
func (s *CohortGroupService) CreateCohortGroup(ctx context.Context, programID uuid.UUID, cohortYear, stage int, groupType CohortGroupType, name string) (*CohortGroup, error) {
	if !ValidCohortGroupType(groupType) {
		return nil, ErrInvalidStatus
	}
	g := &CohortGroup{ProgramID: programID, CohortYear: cohortYear, Stage: stage, Type: groupType, Name: name}
	if err := s.repo.CreateCohortGroup(ctx, g); err != nil {
		return nil, err
	}
	return g, nil
}

// ListCohortGroups returns the cohort groups of one program cohort and stage.
func (s *CohortGroupService) ListCohortGroups(ctx context.Context, programID uuid.UUID, cohortYear, stage int) ([]CohortGroup, error) {
	return s.repo.ListCohortGroups(ctx, programID, cohortYear, stage)
}

// ListCohortGroupsWithCounts returns the cohort groups with member counts.
func (s *CohortGroupService) ListCohortGroupsWithCounts(ctx context.Context, programID uuid.UUID, cohortYear, stage int) ([]CohortGroupWithCount, error) {
	return s.repo.ListCohortGroupsWithCounts(ctx, programID, cohortYear, stage)
}

// AssignToCohortGroup adds a student to a cohort group.
func (s *CohortGroupService) AssignToCohortGroup(ctx context.Context, studentID, groupID uuid.UUID) error {
	exists, err := s.repo.CohortGroupExists(ctx, groupID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrCohortGroupNotFound
	}
	return s.repo.AssignToCohortGroup(ctx, &StudentCohortGroup{StudentID: studentID, CohortGroupID: groupID})
}

// DeleteCohortGroupMember removes a student from a cohort group; removing a
// non-member is a no-op.
func (s *CohortGroupService) DeleteCohortGroupMember(ctx context.Context, studentID, groupID uuid.UUID) error {
	return s.repo.DeleteCohortGroupMember(ctx, studentID, groupID)
}

// StudentCohortGroupIDs returns all cohort groups a student belongs to.
func (s *CohortGroupService) StudentCohortGroupIDs(ctx context.Context, studentID uuid.UUID) ([]uuid.UUID, error) {
	return s.repo.StudentCohortGroupIDs(ctx, studentID)
}

// CohortGroupExists reports whether the cohort group exists.
func (s *CohortGroupService) CohortGroupExists(ctx context.Context, id uuid.UUID) (bool, error) {
	return s.repo.CohortGroupExists(ctx, id)
}

// ReassignCohortGroups moves a student into the least-populated theory and
// practice groups of a cohort. The whole read-pick-write runs in one
// repository transaction (Shape 3), so concurrent reassignments balance
// instead of over-filling the same group.
func (s *CohortGroupService) ReassignCohortGroups(ctx context.Context, studentID, programID uuid.UUID, cohortYear, stage int) error {
	return s.repo.ReassignCohortGroups(ctx, studentID, programID, cohortYear, stage)
}
