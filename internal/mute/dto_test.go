package mute

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestToMuteResponse(t *testing.T) {
	now := time.Now()
	muteID := uuid.New()
	userID := uuid.New()
	mutedBy := uuid.New()
	scopeID := uuid.New()
	reason := "test reason"

	mute := &Mute{
		ID:        muteID,
		UserID:    userID,
		ScopeType: ScopeCourse,
		ScopeID:   &scopeID,
		Reason:    &reason,
		MutedBy:   mutedBy,
		MutedAt:   now,
	}

	resp := ToMuteResponse(mute, now)

	if resp.ID != muteID {
		t.Errorf("ID = %v, want %v", resp.ID, muteID)
	}
	if resp.UserID != userID {
		t.Errorf("UserID = %v, want %v", resp.UserID, userID)
	}
	if resp.ScopeType != ScopeCourse {
		t.Errorf("ScopeType = %v, want %v", resp.ScopeType, ScopeCourse)
	}
	if resp.ScopeID == nil || *resp.ScopeID != scopeID {
		t.Errorf("ScopeID = %v, want %v", resp.ScopeID, scopeID)
	}
	if resp.Reason == nil || *resp.Reason != reason {
		t.Errorf("Reason = %v, want %v", resp.Reason, reason)
	}
	if !resp.IsActive {
		t.Error("IsActive should be true")
	}
}

func TestToMuteResponse_Inactive(t *testing.T) {
	now := time.Now()
	past := now.Add(-1 * time.Hour)

	mute := &Mute{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		ScopeType: ScopeUniversity,
		MutedBy:   uuid.New(),
		MutedAt:   now,
		UnmutedAt: &past,
	}

	resp := ToMuteResponse(mute, now)

	if resp.IsActive {
		t.Error("IsActive should be false for unmuted")
	}
}

func TestToMuteResponse_Expired(t *testing.T) {
	now := time.Now()
	past := now.Add(-1 * time.Hour)

	mute := &Mute{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		ScopeType: ScopeUniversity,
		MutedBy:   uuid.New(),
		MutedAt:   now,
		ExpiresAt: &past,
	}

	resp := ToMuteResponse(mute, now)

	if resp.IsActive {
		t.Error("IsActive should be false for expired")
	}
}

func TestToMuteWithUserResponse(t *testing.T) {
	now := time.Now()
	scopeID := uuid.New()
	offeringName := "Database Systems"
	userNameLocal := "Local Name"

	mute := &MuteWithUser{
		Mute: Mute{
			ID:        uuid.New(),
			UserID:    uuid.New(),
			ScopeType: ScopeCourse,
			ScopeID:   &scopeID,
			MutedBy:   uuid.New(),
			MutedAt:   now,
		},
		UserName:      "John Doe",
		UserNameLocal: &userNameLocal,
		UserEmail:     "john@example.com",
		MutedByName:   "Admin User",
		OfferingName:  &offeringName,
	}

	resp := ToMuteWithUserResponse(mute, now)

	if resp.User == nil {
		t.Fatal("User should not be nil")
	}
	if resp.User.Name != "John Doe" {
		t.Errorf("User.Name = %v, want John Doe", resp.User.Name)
	}
	if resp.User.Email != "john@example.com" {
		t.Errorf("User.Email = %v, want john@example.com", resp.User.Email)
	}
	if resp.User.NameLocal == nil || *resp.User.NameLocal != userNameLocal {
		t.Errorf("User.NameLocal = %v, want %v", resp.User.NameLocal, userNameLocal)
	}

	if resp.MutedByUser == nil {
		t.Fatal("MutedByUser should not be nil")
	}
	if resp.MutedByUser.Name != "Admin User" {
		t.Errorf("MutedByUser.Name = %v, want Admin User", resp.MutedByUser.Name)
	}

	if resp.Offering == nil {
		t.Fatal("Offering should not be nil")
	}
	if resp.Offering.Name != offeringName {
		t.Errorf("Offering.Name = %v, want %v", resp.Offering.Name, offeringName)
	}
}

func TestToMuteResponses(t *testing.T) {
	now := time.Now()
	mutes := []Mute{
		{ID: uuid.New(), UserID: uuid.New(), ScopeType: ScopeCourse, MutedBy: uuid.New(), MutedAt: now},
		{ID: uuid.New(), UserID: uuid.New(), ScopeType: ScopeUniversity, MutedBy: uuid.New(), MutedAt: now},
	}

	responses := ToMuteResponses(mutes, now)

	if len(responses) != 2 {
		t.Errorf("len(responses) = %d, want 2", len(responses))
	}
	if responses[0].ScopeType != ScopeCourse {
		t.Errorf("responses[0].ScopeType = %v, want %v", responses[0].ScopeType, ScopeCourse)
	}
	if responses[1].ScopeType != ScopeUniversity {
		t.Errorf("responses[1].ScopeType = %v, want %v", responses[1].ScopeType, ScopeUniversity)
	}
}

func TestToMuteWithUserResponses(t *testing.T) {
	now := time.Now()
	mutes := []MuteWithUser{
		{
			Mute:        Mute{ID: uuid.New(), UserID: uuid.New(), ScopeType: ScopeCourse, MutedBy: uuid.New(), MutedAt: now},
			UserName:    "User 1",
			UserEmail:   "user1@example.com",
			MutedByName: "Admin",
		},
		{
			Mute:        Mute{ID: uuid.New(), UserID: uuid.New(), ScopeType: ScopeUniversity, MutedBy: uuid.New(), MutedAt: now},
			UserName:    "User 2",
			UserEmail:   "user2@example.com",
			MutedByName: "Admin",
		},
	}

	responses := ToMuteWithUserResponses(mutes, now)

	if len(responses) != 2 {
		t.Errorf("len(responses) = %d, want 2", len(responses))
	}
	if responses[0].User.Name != "User 1" {
		t.Errorf("responses[0].User.Name = %v, want User 1", responses[0].User.Name)
	}
	if responses[1].User.Name != "User 2" {
		t.Errorf("responses[1].User.Name = %v, want User 2", responses[1].User.Name)
	}
}
