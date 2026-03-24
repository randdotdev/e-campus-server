package team

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestToTeamResponse(t *testing.T) {
	now := time.Now()
	teamID := uuid.New()
	leaderID := uuid.New()
	memberID := uuid.New()
	name := "Test Team"

	tests := []struct {
		name  string
		input *TeamWithMembers
		want  TeamResponse
	}{
		{
			name: "with name and members",
			input: &TeamWithMembers{
				Team: Team{
					ID:        teamID,
					Name:      &name,
					LeaderID:  leaderID,
					Status:    StatusActive,
					CreatedAt: now,
				},
				Members: []MemberInfo{
					{StudentID: leaderID, StudentName: "Ali", JoinedAt: now, IsLeader: true},
					{StudentID: memberID, StudentName: "Bob", JoinedAt: now, IsLeader: false},
				},
				MemberCount: 2,
			},
			want: TeamResponse{
				ID:          teamID,
				Name:        "Test Team",
				LeaderID:    leaderID,
				Status:      StatusActive,
				MemberCount: 2,
				Members: []MemberResponse{
					{StudentID: leaderID, StudentName: "Ali", IsLeader: true, JoinedAt: now},
					{StudentID: memberID, StudentName: "Bob", IsLeader: false, JoinedAt: now},
				},
				CreatedAt: now,
			},
		},
		{
			name: "nil name",
			input: &TeamWithMembers{
				Team: Team{
					ID:        teamID,
					Name:      nil,
					LeaderID:  leaderID,
					Status:    StatusActive,
					CreatedAt: now,
				},
				Members:     []MemberInfo{},
				MemberCount: 0,
			},
			want: TeamResponse{
				ID:          teamID,
				Name:        "",
				LeaderID:    leaderID,
				Status:      StatusActive,
				MemberCount: 0,
				Members:     []MemberResponse{},
				CreatedAt:   now,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToTeamResponse(tt.input)
			if got.ID != tt.want.ID {
				t.Errorf("ID = %v, want %v", got.ID, tt.want.ID)
			}
			if got.Name != tt.want.Name {
				t.Errorf("Name = %v, want %v", got.Name, tt.want.Name)
			}
			if got.LeaderID != tt.want.LeaderID {
				t.Errorf("LeaderID = %v, want %v", got.LeaderID, tt.want.LeaderID)
			}
			if got.Status != tt.want.Status {
				t.Errorf("Status = %v, want %v", got.Status, tt.want.Status)
			}
			if got.MemberCount != tt.want.MemberCount {
				t.Errorf("MemberCount = %v, want %v", got.MemberCount, tt.want.MemberCount)
			}
			if len(got.Members) != len(tt.want.Members) {
				t.Errorf("len(Members) = %v, want %v", len(got.Members), len(tt.want.Members))
			}
		})
	}
}

func TestToMyTeamResponse(t *testing.T) {
	now := time.Now()
	teamID := uuid.New()
	leaderID := uuid.New()
	name := "My Team"

	tests := []struct {
		name  string
		input *MyTeam
		want  MyTeamResponse
	}{
		{
			name: "with name",
			input: &MyTeam{
				ID:          teamID,
				Name:        &name,
				LeaderID:    leaderID,
				LeaderName:  "Ali",
				Status:      StatusActive,
				MemberCount: 3,
				IsLeader:    true,
				CreatedAt:   now,
			},
			want: MyTeamResponse{
				ID:          teamID,
				Name:        "My Team",
				LeaderID:    leaderID,
				LeaderName:  "Ali",
				Status:      StatusActive,
				MemberCount: 3,
				IsLeader:    true,
				CreatedAt:   now,
			},
		},
		{
			name: "nil name",
			input: &MyTeam{
				ID:          teamID,
				Name:        nil,
				LeaderID:    leaderID,
				LeaderName:  "Ali",
				Status:      StatusActive,
				MemberCount: 1,
				IsLeader:    false,
				CreatedAt:   now,
			},
			want: MyTeamResponse{
				ID:          teamID,
				Name:        "",
				LeaderID:    leaderID,
				LeaderName:  "Ali",
				Status:      StatusActive,
				MemberCount: 1,
				IsLeader:    false,
				CreatedAt:   now,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToMyTeamResponse(tt.input)
			if got.ID != tt.want.ID {
				t.Errorf("ID = %v, want %v", got.ID, tt.want.ID)
			}
			if got.Name != tt.want.Name {
				t.Errorf("Name = %v, want %v", got.Name, tt.want.Name)
			}
			if got.LeaderName != tt.want.LeaderName {
				t.Errorf("LeaderName = %v, want %v", got.LeaderName, tt.want.LeaderName)
			}
			if got.IsLeader != tt.want.IsLeader {
				t.Errorf("IsLeader = %v, want %v", got.IsLeader, tt.want.IsLeader)
			}
		})
	}
}

func TestToMyTeamsResponse(t *testing.T) {
	now := time.Now()
	name1 := "Team 1"
	name2 := "Team 2"

	teams := []MyTeam{
		{ID: uuid.New(), Name: &name1, LeaderID: uuid.New(), LeaderName: "Ali", Status: StatusActive, MemberCount: 2, IsLeader: true, CreatedAt: now},
		{ID: uuid.New(), Name: &name2, LeaderID: uuid.New(), LeaderName: "Bob", Status: StatusActive, MemberCount: 3, IsLeader: false, CreatedAt: now},
	}

	got := ToMyTeamsResponse(teams)
	if len(got) != 2 {
		t.Fatalf("len = %v, want 2", len(got))
	}
	if got[0].Name != "Team 1" {
		t.Errorf("got[0].Name = %v, want Team 1", got[0].Name)
	}
	if got[1].Name != "Team 2" {
		t.Errorf("got[1].Name = %v, want Team 2", got[1].Name)
	}
}
