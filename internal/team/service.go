package team

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type TeamRepository interface {
	Create(ctx context.Context, t *Team) error
	GetByID(ctx context.Context, id uuid.UUID) (*Team, error)
	Update(ctx context.Context, t *Team) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetByLeader(ctx context.Context, leaderID uuid.UUID) ([]Team, error)
	GetByMember(ctx context.Context, studentID uuid.UUID) ([]MyTeam, error)
	GetWithMembers(ctx context.Context, id uuid.UUID) (*TeamWithMembers, error)
}

type MemberRepository interface {
	Add(ctx context.Context, m *Member) error
	Remove(ctx context.Context, teamID, studentID uuid.UUID) error
	GetMembers(ctx context.Context, teamID uuid.UUID) ([]MemberInfo, error)
	IsMember(ctx context.Context, teamID, studentID uuid.UUID) (bool, error)
	CountMembers(ctx context.Context, teamID uuid.UUID) (int, error)
}

type SubmissionChecker interface {
	TeamHasSubmissions(ctx context.Context, teamID uuid.UUID) (bool, error)
}

type UserProvider interface {
	GetUserName(ctx context.Context, userID uuid.UUID) (string, error)
}

type Service struct {
	teams       TeamRepository
	members     MemberRepository
	submissions SubmissionChecker
	users       UserProvider
}

func NewService(
	teams TeamRepository,
	members MemberRepository,
	submissions SubmissionChecker,
	users UserProvider,
) *Service {
	return &Service{
		teams:       teams,
		members:     members,
		submissions: submissions,
		users:       users,
	}
}

func (s *Service) Create(ctx context.Context, leaderID uuid.UUID, name *string) (*Team, error) {
	var teamName *string
	if name != nil && *name != "" {
		teamName = name
	} else {
		userName, err := s.users.GetUserName(ctx, leaderID)
		if err != nil {
			return nil, err
		}
		defaultName := GetDefaultTeamName(userName)
		teamName = &defaultName
	}

	team := &Team{
		ID:        uuid.New(),
		Name:      teamName,
		LeaderID:  leaderID,
		Status:    StatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.teams.Create(ctx, team); err != nil {
		return nil, err
	}

	member := &Member{
		ID:        uuid.New(),
		TeamID:    team.ID,
		StudentID: leaderID,
		JoinedAt:  time.Now(),
	}
	if err := s.members.Add(ctx, member); err != nil {
		return nil, err
	}

	return team, nil
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*TeamWithMembers, error) {
	team, err := s.teams.GetWithMembers(ctx, id)
	if err != nil {
		return nil, err
	}
	if team == nil {
		return nil, ErrTeamNotFound
	}
	return team, nil
}

func (s *Service) GetByIDForUser(ctx context.Context, id, userID uuid.UUID) (*TeamWithMembers, error) {
	team, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if team.LeaderID == userID {
		return team, nil
	}
	isMember, err := s.members.IsMember(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, ErrTeamNotFound
	}
	return team, nil
}

func (s *Service) GetMyTeams(ctx context.Context, studentID uuid.UUID, status *string) ([]MyTeam, error) {
	teams, err := s.teams.GetByMember(ctx, studentID)
	if err != nil {
		return nil, err
	}

	for i := range teams {
		teams[i].IsLeader = teams[i].LeaderID == studentID
	}

	if status != nil {
		filtered := make([]MyTeam, 0)
		for _, t := range teams {
			if t.Status == *status {
				filtered = append(filtered, t)
			}
		}
		return filtered, nil
	}

	return teams, nil
}

func (s *Service) UpdateName(ctx context.Context, teamID, userID uuid.UUID, name string) (*Team, error) {
	team, err := s.teams.GetByID(ctx, teamID)
	if err != nil {
		return nil, err
	}
	if team == nil {
		return nil, ErrTeamNotFound
	}

	if !IsLeader(team, userID) {
		return nil, ErrNotLeader
	}

	team.Name = &name
	team.UpdatedAt = time.Now()

	if err := s.teams.Update(ctx, team); err != nil {
		return nil, err
	}

	return team, nil
}

func (s *Service) AddMember(ctx context.Context, teamID, leaderID, studentID uuid.UUID) error {
	team, err := s.teams.GetByID(ctx, teamID)
	if err != nil {
		return err
	}
	if team == nil {
		return ErrTeamNotFound
	}

	if !IsLeader(team, leaderID) {
		return ErrNotLeader
	}

	if !IsActive(team.Status) {
		return ErrTeamArchived
	}

	hasSubmissions, err := s.submissions.TeamHasSubmissions(ctx, teamID)
	if err != nil {
		return err
	}
	if hasSubmissions {
		return ErrTeamLocked
	}

	isMember, err := s.members.IsMember(ctx, teamID, studentID)
	if err != nil {
		return err
	}
	if isMember {
		return ErrAlreadyMember
	}

	count, err := s.members.CountMembers(ctx, teamID)
	if err != nil {
		return err
	}
	if count >= MaxMembers {
		return ErrMaxMembers
	}

	member := &Member{
		ID:        uuid.New(),
		TeamID:    teamID,
		StudentID: studentID,
		JoinedAt:  time.Now(),
	}

	return s.members.Add(ctx, member)
}

func (s *Service) RemoveMember(ctx context.Context, teamID, leaderID, studentID uuid.UUID) error {
	team, err := s.teams.GetByID(ctx, teamID)
	if err != nil {
		return err
	}
	if team == nil {
		return ErrTeamNotFound
	}

	if !IsLeader(team, leaderID) {
		return ErrNotLeader
	}

	if studentID == team.LeaderID {
		return ErrCannotRemoveLeader
	}

	hasSubmissions, err := s.submissions.TeamHasSubmissions(ctx, teamID)
	if err != nil {
		return err
	}
	if hasSubmissions {
		return ErrTeamLocked
	}

	isMember, err := s.members.IsMember(ctx, teamID, studentID)
	if err != nil {
		return err
	}
	if !isMember {
		return ErrMemberNotFound
	}

	return s.members.Remove(ctx, teamID, studentID)
}

func (s *Service) Leave(ctx context.Context, teamID, studentID uuid.UUID) error {
	team, err := s.teams.GetByID(ctx, teamID)
	if err != nil {
		return err
	}
	if team == nil {
		return ErrTeamNotFound
	}

	if studentID == team.LeaderID {
		return ErrLeaderCannotLeave
	}

	hasSubmissions, err := s.submissions.TeamHasSubmissions(ctx, teamID)
	if err != nil {
		return err
	}
	if hasSubmissions {
		return ErrTeamLocked
	}

	isMember, err := s.members.IsMember(ctx, teamID, studentID)
	if err != nil {
		return err
	}
	if !isMember {
		return ErrNotMember
	}

	return s.members.Remove(ctx, teamID, studentID)
}

func (s *Service) TransferLeadership(ctx context.Context, teamID, currentLeaderID, newLeaderID uuid.UUID) (*Team, error) {
	team, err := s.teams.GetByID(ctx, teamID)
	if err != nil {
		return nil, err
	}
	if team == nil {
		return nil, ErrTeamNotFound
	}

	if !IsLeader(team, currentLeaderID) {
		return nil, ErrNotLeader
	}

	isMember, err := s.members.IsMember(ctx, teamID, newLeaderID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, ErrNotMember
	}

	team.LeaderID = newLeaderID
	team.UpdatedAt = time.Now()

	if err := s.teams.Update(ctx, team); err != nil {
		return nil, err
	}

	return team, nil
}

func (s *Service) Delete(ctx context.Context, teamID, leaderID uuid.UUID) error {
	team, err := s.teams.GetByID(ctx, teamID)
	if err != nil {
		return err
	}
	if team == nil {
		return ErrTeamNotFound
	}

	if !IsLeader(team, leaderID) {
		return ErrNotLeader
	}

	hasSubmissions, err := s.submissions.TeamHasSubmissions(ctx, teamID)
	if err != nil {
		return err
	}
	if hasSubmissions {
		return ErrTeamLocked
	}

	return s.teams.Delete(ctx, teamID)
}

func (s *Service) Archive(ctx context.Context, teamID uuid.UUID) error {
	team, err := s.teams.GetByID(ctx, teamID)
	if err != nil {
		return err
	}
	if team == nil {
		return ErrTeamNotFound
	}

	if team.Status == StatusArchived {
		return nil
	}

	team.Status = StatusArchived
	team.UpdatedAt = time.Now()

	return s.teams.Update(ctx, team)
}

func (s *Service) ArchiveTeamsForStudent(ctx context.Context, studentID uuid.UUID) error {
	teams, err := s.teams.GetByMember(ctx, studentID)
	if err != nil {
		return err
	}

	for _, t := range teams {
		if t.Status == StatusActive {
			if err := s.Archive(ctx, t.ID); err != nil {
				return err
			}
		}
	}

	return nil
}
