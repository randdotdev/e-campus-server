package mute

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestIsMuteActive(t *testing.T) {
	now := time.Now()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	tests := []struct {
		name string
		mute *Mute
		want bool
	}{
		{"nil mute", nil, false},
		{"active mute no expiry", &Mute{}, true},
		{"active mute future expiry", &Mute{ExpiresAt: &future}, true},
		{"expired mute", &Mute{ExpiresAt: &past}, false},
		{"unmuted", &Mute{UnmutedAt: &past}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsMuteActive(tt.mute, now); got != tt.want {
				t.Errorf("IsMuteActive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsMuteExpired(t *testing.T) {
	now := time.Now()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	tests := []struct {
		name string
		mute *Mute
		want bool
	}{
		{"nil mute", nil, false},
		{"no expiry", &Mute{}, false},
		{"future expiry", &Mute{ExpiresAt: &future}, false},
		{"past expiry", &Mute{ExpiresAt: &past}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsMuteExpired(tt.mute, now); got != tt.want {
				t.Errorf("IsMuteExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCanMuteUser(t *testing.T) {
	userA := uuid.New()
	userB := uuid.New()

	tests := []struct {
		name    string
		actor   uuid.UUID
		target  uuid.UUID
		wantErr error
	}{
		{"different users", userA, userB, nil},
		{"same user", userA, userA, ErrCannotMuteSelf},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CanMuteUser(tt.actor, tt.target)
			if err != tt.wantErr {
				t.Errorf("CanMuteUser() = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestBuildMute(t *testing.T) {
	userID := uuid.New()
	mutedBy := uuid.New()
	scopeID := uuid.New()
	reason := "test reason"
	expiresAt := time.Now().Add(24 * time.Hour)

	mute := BuildMute(userID, ScopeCourse, &scopeID, &reason, mutedBy, &expiresAt)

	if mute.ID == uuid.Nil {
		t.Error("ID should be generated")
	}
	if mute.UserID != userID {
		t.Errorf("UserID = %v, want %v", mute.UserID, userID)
	}
	if mute.ScopeType != ScopeCourse {
		t.Errorf("ScopeType = %v, want %v", mute.ScopeType, ScopeCourse)
	}
	if mute.ScopeID == nil || *mute.ScopeID != scopeID {
		t.Errorf("ScopeID = %v, want %v", mute.ScopeID, scopeID)
	}
	if mute.Reason == nil || *mute.Reason != reason {
		t.Errorf("Reason = %v, want %v", mute.Reason, reason)
	}
	if mute.MutedBy != mutedBy {
		t.Errorf("MutedBy = %v, want %v", mute.MutedBy, mutedBy)
	}
}

func TestValidateScopeType(t *testing.T) {
	tests := []struct {
		name      string
		scopeType string
		want      bool
	}{
		{"course", ScopeCourse, true},
		{"university", ScopeUniversity, true},
		{"invalid", "invalid", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateScopeType(tt.scopeType); got != tt.want {
				t.Errorf("ValidateScopeType(%v) = %v, want %v", tt.scopeType, got, tt.want)
			}
		})
	}
}

func TestValidateMuteScope(t *testing.T) {
	scopeID := uuid.New()

	tests := []struct {
		name      string
		scopeType string
		scopeID   *uuid.UUID
		want      bool
	}{
		{"university with nil", ScopeUniversity, nil, true},
		{"university with id", ScopeUniversity, &scopeID, false},
		{"course with id", ScopeCourse, &scopeID, true},
		{"course with nil", ScopeCourse, nil, false},
		{"invalid type", "invalid", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateMuteScope(tt.scopeType, tt.scopeID); got != tt.want {
				t.Errorf("ValidateMuteScope(%v, %v) = %v, want %v", tt.scopeType, tt.scopeID, got, tt.want)
			}
		})
	}
}
