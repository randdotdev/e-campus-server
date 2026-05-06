package team

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

type mockTeamRepo struct {
	teams map[uuid.UUID]*Team
}

func newMockTeamRepo() *mockTeamRepo {
	return &mockTeamRepo{teams: make(map[uuid.UUID]*Team)}
}

func (m *mockTeamRepo) Create(_ context.Context, t *Team) error {
	m.teams[t.ID] = t
	return nil
}

func (m *mockTeamRepo) GetByID(_ context.Context, id uuid.UUID) (*Team, error) {
	return m.teams[id], nil
}

func (m *mockTeamRepo) Update(_ context.Context, t *Team) error {
	m.teams[t.ID] = t
	return nil
}

func (m *mockTeamRepo) Delete(_ context.Context, id uuid.UUID) error {
	delete(m.teams, id)
	return nil
}

func (m *mockTeamRepo) GetByLeader(_ context.Context, leaderID uuid.UUID) ([]Team, error) {
	var result []Team
	for _, t := range m.teams {
		if t.LeaderID == leaderID {
			result = append(result, *t)
		}
	}
	return result, nil
}

func (m *mockTeamRepo) GetByMember(_ context.Context, _ uuid.UUID) ([]MyTeam, error) {
	return []MyTeam{}, nil
}

func (m *mockTeamRepo) GetWithMembers(ctx context.Context, id uuid.UUID) (*TeamWithMembers, error) {
	t := m.teams[id]
	if t == nil {
		return nil, nil
	}
	return &TeamWithMembers{Team: *t, Members: []MemberInfo{}, MemberCount: 1}, nil
}

type mockMemberRepo struct {
	members map[uuid.UUID][]uuid.UUID
}

func newMockMemberRepo() *mockMemberRepo {
	return &mockMemberRepo{members: make(map[uuid.UUID][]uuid.UUID)}
}

func (m *mockMemberRepo) Add(_ context.Context, member *Member) error {
	m.members[member.TeamID] = append(m.members[member.TeamID], member.StudentID)
	return nil
}

func (m *mockMemberRepo) Remove(_ context.Context, teamID, studentID uuid.UUID) error {
	members := m.members[teamID]
	for i, id := range members {
		if id == studentID {
			m.members[teamID] = append(members[:i], members[i+1:]...)
			break
		}
	}
	return nil
}

func (m *mockMemberRepo) GetMembers(_ context.Context, teamID uuid.UUID) ([]MemberInfo, error) {
	var result []MemberInfo
	for _, id := range m.members[teamID] {
		result = append(result, MemberInfo{StudentID: id})
	}
	return result, nil
}

func (m *mockMemberRepo) IsMember(_ context.Context, teamID, studentID uuid.UUID) (bool, error) {
	for _, id := range m.members[teamID] {
		if id == studentID {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockMemberRepo) CountMembers(_ context.Context, teamID uuid.UUID) (int, error) {
	return len(m.members[teamID]), nil
}

type mockSubmissionChecker struct {
	hasSubmissions bool
}

func (m *mockSubmissionChecker) TeamHasSubmissions(_ context.Context, _ uuid.UUID) (bool, error) {
	return m.hasSubmissions, nil
}

type mockUserProvider struct {
	name string
}

func (m *mockUserProvider) GetUserName(_ context.Context, _ uuid.UUID) (string, error) {
	return m.name, nil
}

func TestService_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("success with name", func(t *testing.T) {
		teamRepo := newMockTeamRepo()
		memberRepo := newMockMemberRepo()
		service := NewService(teamRepo, memberRepo, &mockSubmissionChecker{}, &mockUserProvider{name: "Ali"})

		name := "My Team"
		team, err := service.Create(ctx, uuid.New(), &name)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if *team.Name != "My Team" {
			t.Errorf("Name = %v, want My Team", *team.Name)
		}
	})

	t.Run("success with default name", func(t *testing.T) {
		teamRepo := newMockTeamRepo()
		memberRepo := newMockMemberRepo()
		service := NewService(teamRepo, memberRepo, &mockSubmissionChecker{}, &mockUserProvider{name: "Ali"})

		team, err := service.Create(ctx, uuid.New(), nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if *team.Name != "Ali's Team" {
			t.Errorf("Name = %v, want Ali's Team", *team.Name)
		}
	})
}

func TestService_AddMember(t *testing.T) {
	ctx := context.Background()
	leaderID := uuid.New()
	memberID := uuid.New()

	t.Run("success", func(t *testing.T) {
		teamRepo := newMockTeamRepo()
		memberRepo := newMockMemberRepo()
		service := NewService(teamRepo, memberRepo, &mockSubmissionChecker{}, &mockUserProvider{name: "Ali"})

		team, _ := service.Create(ctx, leaderID, nil)
		err := service.AddMember(ctx, team.ID, leaderID, memberID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("not leader", func(t *testing.T) {
		teamRepo := newMockTeamRepo()
		memberRepo := newMockMemberRepo()
		service := NewService(teamRepo, memberRepo, &mockSubmissionChecker{}, &mockUserProvider{name: "Ali"})

		team, _ := service.Create(ctx, leaderID, nil)
		err := service.AddMember(ctx, team.ID, memberID, uuid.New())
		if !errors.Is(err, ErrNotLeader) {
			t.Errorf("expected ErrNotLeader, got %v", err)
		}
	})

	t.Run("team locked", func(t *testing.T) {
		teamRepo := newMockTeamRepo()
		memberRepo := newMockMemberRepo()
		service := NewService(teamRepo, memberRepo, &mockSubmissionChecker{hasSubmissions: true}, &mockUserProvider{name: "Ali"})

		team, _ := service.Create(ctx, leaderID, nil)
		err := service.AddMember(ctx, team.ID, leaderID, memberID)
		if !errors.Is(err, ErrTeamLocked) {
			t.Errorf("expected ErrTeamLocked, got %v", err)
		}
	})

	t.Run("already member", func(t *testing.T) {
		teamRepo := newMockTeamRepo()
		memberRepo := newMockMemberRepo()
		service := NewService(teamRepo, memberRepo, &mockSubmissionChecker{}, &mockUserProvider{name: "Ali"})

		team, _ := service.Create(ctx, leaderID, nil)
		_ = service.AddMember(ctx, team.ID, leaderID, memberID)
		err := service.AddMember(ctx, team.ID, leaderID, memberID)
		if !errors.Is(err, ErrAlreadyMember) {
			t.Errorf("expected ErrAlreadyMember, got %v", err)
		}
	})
}

func TestService_RemoveMember(t *testing.T) {
	ctx := context.Background()
	leaderID := uuid.New()
	memberID := uuid.New()

	t.Run("success", func(t *testing.T) {
		teamRepo := newMockTeamRepo()
		memberRepo := newMockMemberRepo()
		service := NewService(teamRepo, memberRepo, &mockSubmissionChecker{}, &mockUserProvider{name: "Ali"})

		team, _ := service.Create(ctx, leaderID, nil)
		_ = service.AddMember(ctx, team.ID, leaderID, memberID)
		err := service.RemoveMember(ctx, team.ID, leaderID, memberID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("cannot remove leader", func(t *testing.T) {
		teamRepo := newMockTeamRepo()
		memberRepo := newMockMemberRepo()
		service := NewService(teamRepo, memberRepo, &mockSubmissionChecker{}, &mockUserProvider{name: "Ali"})

		team, _ := service.Create(ctx, leaderID, nil)
		err := service.RemoveMember(ctx, team.ID, leaderID, leaderID)
		if !errors.Is(err, ErrCannotRemoveLeader) {
			t.Errorf("expected ErrCannotRemoveLeader, got %v", err)
		}
	})
}

func TestService_Leave(t *testing.T) {
	ctx := context.Background()
	leaderID := uuid.New()
	memberID := uuid.New()

	t.Run("success", func(t *testing.T) {
		teamRepo := newMockTeamRepo()
		memberRepo := newMockMemberRepo()
		service := NewService(teamRepo, memberRepo, &mockSubmissionChecker{}, &mockUserProvider{name: "Ali"})

		team, _ := service.Create(ctx, leaderID, nil)
		_ = service.AddMember(ctx, team.ID, leaderID, memberID)
		err := service.Leave(ctx, team.ID, memberID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("leader cannot leave", func(t *testing.T) {
		teamRepo := newMockTeamRepo()
		memberRepo := newMockMemberRepo()
		service := NewService(teamRepo, memberRepo, &mockSubmissionChecker{}, &mockUserProvider{name: "Ali"})

		team, _ := service.Create(ctx, leaderID, nil)
		err := service.Leave(ctx, team.ID, leaderID)
		if !errors.Is(err, ErrLeaderCannotLeave) {
			t.Errorf("expected ErrLeaderCannotLeave, got %v", err)
		}
	})
}

func TestService_TransferLeadership(t *testing.T) {
	ctx := context.Background()
	leaderID := uuid.New()
	newLeaderID := uuid.New()

	t.Run("success", func(t *testing.T) {
		teamRepo := newMockTeamRepo()
		memberRepo := newMockMemberRepo()
		service := NewService(teamRepo, memberRepo, &mockSubmissionChecker{}, &mockUserProvider{name: "Ali"})

		team, _ := service.Create(ctx, leaderID, nil)
		_ = service.AddMember(ctx, team.ID, leaderID, newLeaderID)

		updated, err := service.TransferLeadership(ctx, team.ID, leaderID, newLeaderID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if updated.LeaderID != newLeaderID {
			t.Errorf("LeaderID = %v, want %v", updated.LeaderID, newLeaderID)
		}
	})

	t.Run("not leader", func(t *testing.T) {
		teamRepo := newMockTeamRepo()
		memberRepo := newMockMemberRepo()
		service := NewService(teamRepo, memberRepo, &mockSubmissionChecker{}, &mockUserProvider{name: "Ali"})

		team, _ := service.Create(ctx, leaderID, nil)
		_, err := service.TransferLeadership(ctx, team.ID, newLeaderID, uuid.New())
		if !errors.Is(err, ErrNotLeader) {
			t.Errorf("expected ErrNotLeader, got %v", err)
		}
	})

	t.Run("new leader not member", func(t *testing.T) {
		teamRepo := newMockTeamRepo()
		memberRepo := newMockMemberRepo()
		service := NewService(teamRepo, memberRepo, &mockSubmissionChecker{}, &mockUserProvider{name: "Ali"})

		team, _ := service.Create(ctx, leaderID, nil)
		_, err := service.TransferLeadership(ctx, team.ID, leaderID, newLeaderID)
		if !errors.Is(err, ErrNotMember) {
			t.Errorf("expected ErrNotMember, got %v", err)
		}
	})
}

func TestService_Delete(t *testing.T) {
	ctx := context.Background()
	leaderID := uuid.New()

	t.Run("success", func(t *testing.T) {
		teamRepo := newMockTeamRepo()
		memberRepo := newMockMemberRepo()
		service := NewService(teamRepo, memberRepo, &mockSubmissionChecker{}, &mockUserProvider{name: "Ali"})

		team, _ := service.Create(ctx, leaderID, nil)
		err := service.Delete(ctx, team.ID, leaderID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("not leader", func(t *testing.T) {
		teamRepo := newMockTeamRepo()
		memberRepo := newMockMemberRepo()
		service := NewService(teamRepo, memberRepo, &mockSubmissionChecker{}, &mockUserProvider{name: "Ali"})

		team, _ := service.Create(ctx, leaderID, nil)
		err := service.Delete(ctx, team.ID, uuid.New())
		if !errors.Is(err, ErrNotLeader) {
			t.Errorf("expected ErrNotLeader, got %v", err)
		}
	})

	t.Run("team locked", func(t *testing.T) {
		teamRepo := newMockTeamRepo()
		memberRepo := newMockMemberRepo()
		service := NewService(teamRepo, memberRepo, &mockSubmissionChecker{hasSubmissions: true}, &mockUserProvider{name: "Ali"})

		team, _ := service.Create(ctx, leaderID, nil)
		err := service.Delete(ctx, team.ID, leaderID)
		if !errors.Is(err, ErrTeamLocked) {
			t.Errorf("expected ErrTeamLocked, got %v", err)
		}
	})
}
