package team

import (
	"testing"

	"github.com/google/uuid"
)

func TestIsLeader(t *testing.T) {
	leaderID := uuid.New()
	otherID := uuid.New()

	team := &Team{LeaderID: leaderID}

	tests := []struct {
		name   string
		userID uuid.UUID
		want   bool
	}{
		{"leader matches", leaderID, true},
		{"leader does not match", otherID, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsLeader(team, tt.userID); got != tt.want {
				t.Errorf("IsLeader() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsActive(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{"active status", StatusActive, true},
		{"archived status", StatusArchived, false},
		{"invalid status", "invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsActive(tt.status); got != tt.want {
				t.Errorf("IsActive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidStatus(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{"active valid", StatusActive, true},
		{"archived valid", StatusArchived, true},
		{"invalid status", "pending", false},
		{"empty status", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidStatus(tt.status); got != tt.want {
				t.Errorf("IsValidStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetDefaultTeamName(t *testing.T) {
	tests := []struct {
		name       string
		leaderName string
		want       string
	}{
		{"simple name", "Ali", "Ali's Team"},
		{"full name", "John Doe", "John Doe's Team"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetDefaultTeamName(tt.leaderName); got != tt.want {
				t.Errorf("GetDefaultTeamName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCanModifyMembers(t *testing.T) {
	tests := []struct {
		name           string
		status         string
		hasSubmissions bool
		want           bool
	}{
		{"active no submissions", StatusActive, false, true},
		{"active has submissions", StatusActive, true, false},
		{"archived no submissions", StatusArchived, false, false},
		{"archived has submissions", StatusArchived, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CanModifyMembers(tt.status, tt.hasSubmissions); got != tt.want {
				t.Errorf("CanModifyMembers() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidMemberCount(t *testing.T) {
	tests := []struct {
		name  string
		count int
		want  bool
	}{
		{"zero invalid", 0, false},
		{"one valid", 1, true},
		{"five valid", 5, true},
		{"max valid", MaxMembers, true},
		{"over max invalid", MaxMembers + 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidMemberCount(tt.count); got != tt.want {
				t.Errorf("IsValidMemberCount() = %v, want %v", got, tt.want)
			}
		})
	}
}
