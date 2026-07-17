package classroom

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// A team is a student-formed group that outlives any single project — the
// same friends register for project after project. A team binds to its
// creator's program and cohort at creation, and only students of that same
// program and cohort may join: classmates group with classmates, and
// nobody can be pulled into a stranger's team. The leader manages
// membership and may hand leadership over; a team that has carried a
// submission is locked (its membership is historical record), and an
// archived team is read-only.

// ── Value objects ───────────────────────────────────────────────────────────

type TeamStatus string

const (
	TeamActive   TeamStatus = "active"
	TeamArchived TeamStatus = "archived"
)

func ValidTeamStatus(s TeamStatus) bool { return s == TeamActive || s == TeamArchived }

// MaxTeamMembers caps a team independent of any project's own bounds.
const MaxTeamMembers = 10

// ── Entities ────────────────────────────────────────────────────────────────

type Team struct {
	ID         uuid.UUID  `db:"id"`
	Name       *string    `db:"name"`
	LeaderID   uuid.UUID  `db:"leader_id"`
	ProgramID  uuid.UUID  `db:"program_id"`
	CohortYear int        `db:"cohort_year"`
	Status     TeamStatus `db:"status"`
	Version    int64      `db:"version"`
	CreatedAt  time.Time  `db:"created_at"`
	UpdatedAt  time.Time  `db:"updated_at"`
}

type TeamMember struct {
	ID       uuid.UUID `db:"id"`
	TeamID   uuid.UUID `db:"team_id"`
	UserID   uuid.UUID `db:"student_id"`
	JoinedAt time.Time `db:"joined_at"`
}

// ── Derived read models ─────────────────────────────────────────────────────

// TeamMemberInfo joins member display columns (team_members ⋈ users).
type TeamMemberInfo struct {
	UserID   uuid.UUID `db:"user_id"`
	Name     string    `db:"name"`
	Email    string    `db:"email"`
	Avatar   *string   `db:"avatar"`
	JoinedAt time.Time `db:"joined_at"`
}

// TeamWithMembers is the full team view.
type TeamWithMembers struct {
	Team
	Members []TeamMemberInfo
}

// MyTeam is one row of the caller's team list, with their own standing.
type MyTeam struct {
	Team
	MemberCount int  `db:"member_count"`
	IsLeader    bool `db:"-"`
}

// ── Rules ───────────────────────────────────────────────────────────────────

// DefaultTeamName derives a name for teams created without one.
func DefaultTeamName(leaderName string) string {
	return leaderName + "'s Team"
}

// ── Ports ───────────────────────────────────────────────────────────────────

// TeamRepository persists teams and members.
//
// AddMember inserts under the team's own guards, all in the statement:
// team active, not locked by submissions, member cap not exceeded, and the
// member a student of the team's program and cohort — a cap or lock race
// cannot over-fill. Its misses map to ErrTeamArchived, ErrTeamLocked,
// ErrTeamFull, ErrNotClassmate, ErrAlreadyMember. RemoveMember carries the
// same lock guard. UpdateTeam is a version compare-and-swap. DeleteTeam
// refuses a team with submissions. TeamHasSubmissions feeds the lock rule
// for pre-check messages.
type TeamRepository interface {
	CreateTeam(ctx context.Context, t *Team, leader *TeamMember) error
	GetTeam(ctx context.Context, id uuid.UUID) (*Team, error)
	GetTeamWithMembers(ctx context.Context, id uuid.UUID) (*TeamWithMembers, error)
	ListMyTeams(ctx context.Context, userID uuid.UUID) ([]MyTeam, error)
	UpdateTeam(ctx context.Context, t *Team, expectedVersion int64) (int64, error)
	DeleteTeam(ctx context.Context, id uuid.UUID) error

	AddMember(ctx context.Context, m *TeamMember, maxMembers int) error
	RemoveMember(ctx context.Context, teamID, userID uuid.UUID) error
	IsMember(ctx context.Context, teamID, userID uuid.UUID) (bool, error)
	TeamMemberIDs(ctx context.Context, teamID uuid.UUID) ([]uuid.UUID, error)
	TeamHasSubmissions(ctx context.Context, teamID uuid.UUID) (bool, error)
}

// ── Service ─────────────────────────────────────────────────────────────────

// TeamService manages teams. Leadership checks live here — the gate knows
// offerings, not teams.
type TeamService struct {
	repo     TeamRepository
	users    UserReader
	students StudentReader
}

func NewTeamService(repo TeamRepository, users UserReader, students StudentReader) *TeamService {
	return &TeamService{repo: repo, users: users, students: students}
}

// Create starts a team with the caller as leader and first member, bound
// to the caller's program and cohort; non-students cannot create teams.
func (s *TeamService) Create(ctx context.Context, leaderID uuid.UUID, name *string) (*Team, error) {
	programID, cohortYear, err := s.students.StudentProgramCohort(ctx, leaderID)
	if err != nil {
		return nil, err
	}
	teamName := name
	if teamName == nil || *teamName == "" {
		leaderName, err := s.users.UserName(ctx, leaderID)
		if err != nil {
			return nil, err
		}
		n := DefaultTeamName(leaderName)
		teamName = &n
	}
	now := time.Now()
	t := &Team{
		ID:         uuid.New(),
		Name:       teamName,
		LeaderID:   leaderID,
		ProgramID:  programID,
		CohortYear: cohortYear,
		Status:     TeamActive,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	leader := &TeamMember{ID: uuid.New(), TeamID: t.ID, UserID: leaderID, JoinedAt: now}
	if err := s.repo.CreateTeam(ctx, t, leader); err != nil {
		return nil, err
	}
	return t, nil
}

// Get returns the team to its own members and leader only.
func (s *TeamService) Get(ctx context.Context, id, readerID uuid.UUID) (*TeamWithMembers, error) {
	t, err := s.repo.GetTeamWithMembers(ctx, id)
	if err != nil {
		return nil, err
	}
	if t.LeaderID != readerID {
		isMember, err := s.repo.IsMember(ctx, id, readerID)
		if err != nil {
			return nil, err
		}
		if !isMember {
			return nil, ErrTeamNotFound
		}
	}
	return t, nil
}

// MyTeams lists the caller's teams, optionally narrowed by status.
func (s *TeamService) MyTeams(ctx context.Context, userID uuid.UUID, status *TeamStatus) ([]MyTeam, error) {
	if status != nil && !ValidTeamStatus(*status) {
		return nil, ErrInvalidInput
	}
	teams, err := s.repo.ListMyTeams(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := make([]MyTeam, 0, len(teams))
	for _, t := range teams {
		if status != nil && t.Status != *status {
			continue
		}
		t.IsLeader = t.LeaderID == userID
		result = append(result, t)
	}
	return result, nil
}

// Rename is leader-only.
func (s *TeamService) Rename(ctx context.Context, teamID, actorID uuid.UUID, name string) (*Team, error) {
	if name == "" {
		return nil, ErrInvalidInput
	}
	t, err := s.leaderTeam(ctx, teamID, actorID)
	if err != nil {
		return nil, err
	}
	t.Name = &name
	t.UpdatedAt = time.Now()
	newVersion, err := s.repo.UpdateTeam(ctx, t, t.Version)
	if err != nil {
		return nil, err
	}
	t.Version = newVersion
	return t, nil
}

// AddMember is leader-only; the cap, lock, and archive guards are in the
// insert itself.
func (s *TeamService) AddMember(ctx context.Context, teamID, actorID, userID uuid.UUID) error {
	if _, err := s.leaderTeam(ctx, teamID, actorID); err != nil {
		return err
	}
	m := &TeamMember{ID: uuid.New(), TeamID: teamID, UserID: userID, JoinedAt: time.Now()}
	return s.repo.AddMember(ctx, m, MaxTeamMembers)
}

// RemoveMember is leader-only; the leader removes others, never themself.
func (s *TeamService) RemoveMember(ctx context.Context, teamID, actorID, userID uuid.UUID) error {
	t, err := s.leaderTeam(ctx, teamID, actorID)
	if err != nil {
		return err
	}
	if userID == t.LeaderID {
		return ErrLeaderCannotLeave
	}
	return s.repo.RemoveMember(ctx, teamID, userID)
}

// Leave removes the caller; a leader transfers leadership first.
func (s *TeamService) Leave(ctx context.Context, teamID, userID uuid.UUID) error {
	t, err := s.repo.GetTeam(ctx, teamID)
	if err != nil {
		return err
	}
	if t.LeaderID == userID {
		return ErrLeaderCannotLeave
	}
	return s.repo.RemoveMember(ctx, teamID, userID)
}

// TransferLeadership hands the team to another member.
func (s *TeamService) TransferLeadership(ctx context.Context, teamID, actorID, newLeaderID uuid.UUID) (*Team, error) {
	t, err := s.leaderTeam(ctx, teamID, actorID)
	if err != nil {
		return nil, err
	}
	isMember, err := s.repo.IsMember(ctx, teamID, newLeaderID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, ErrNotMember
	}
	t.LeaderID = newLeaderID
	t.UpdatedAt = time.Now()
	newVersion, err := s.repo.UpdateTeam(ctx, t, t.Version)
	if err != nil {
		return nil, err
	}
	t.Version = newVersion
	return t, nil
}

// Delete is leader-only; the submission lock is enforced in the statement.
func (s *TeamService) Delete(ctx context.Context, teamID, actorID uuid.UUID) error {
	if _, err := s.leaderTeam(ctx, teamID, actorID); err != nil {
		return err
	}
	return s.repo.DeleteTeam(ctx, teamID)
}

func (s *TeamService) leaderTeam(ctx context.Context, teamID, actorID uuid.UUID) (*Team, error) {
	t, err := s.repo.GetTeam(ctx, teamID)
	if err != nil {
		return nil, err
	}
	if t.LeaderID != actorID {
		return nil, ErrNotLeader
	}
	return t, nil
}
