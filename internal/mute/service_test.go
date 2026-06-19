package mute

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/randdotdev/e-campus-server/internal/pagination"
)

// Mock implementations

type mockMuteRepo struct {
	mutes map[uuid.UUID]*Mute
}

func newMockMuteRepo() *mockMuteRepo {
	return &mockMuteRepo{mutes: make(map[uuid.UUID]*Mute)}
}

func (m *mockMuteRepo) Create(ctx context.Context, mute *Mute) error {
	m.mutes[mute.ID] = mute
	return nil
}

func (m *mockMuteRepo) GetByID(ctx context.Context, id uuid.UUID) (*Mute, error) {
	return m.mutes[id], nil
}

func (m *mockMuteRepo) GetActiveMute(ctx context.Context, userID uuid.UUID, scopeType string, scopeID *uuid.UUID) (*Mute, error) {
	for _, mute := range m.mutes {
		if mute.UserID != userID || mute.ScopeType != scopeType {
			continue
		}
		if scopeID == nil && mute.ScopeID == nil {
			if mute.UnmutedAt == nil {
				return mute, nil
			}
		}
		if scopeID != nil && mute.ScopeID != nil && *scopeID == *mute.ScopeID {
			if mute.UnmutedAt == nil {
				return mute, nil
			}
		}
	}
	return nil, nil
}

func (m *mockMuteRepo) IsMuted(ctx context.Context, userID uuid.UUID, offeringID *uuid.UUID) (bool, error) {
	for _, mute := range m.mutes {
		if mute.UserID != userID || mute.UnmutedAt != nil {
			continue
		}
		if mute.ScopeType == ScopeUniversity {
			return true, nil
		}
		if mute.ScopeType == ScopeCourse && offeringID != nil && mute.ScopeID != nil && *mute.ScopeID == *offeringID {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockMuteRepo) Unmute(ctx context.Context, id uuid.UUID, unmutedBy uuid.UUID) error {
	mute, exists := m.mutes[id]
	if !exists || mute.UnmutedAt != nil {
		return ErrMuteNotFound
	}
	now := time.Now()
	mute.UnmutedAt = &now
	mute.UnmutedBy = &unmutedBy
	return nil
}

func (m *mockMuteRepo) UnmuteAll(ctx context.Context, userID uuid.UUID, unmutedBy uuid.UUID) (int64, error) {
	var count int64
	now := time.Now()
	for _, mute := range m.mutes {
		if mute.UserID == userID && mute.UnmutedAt == nil {
			mute.UnmutedAt = &now
			mute.UnmutedBy = &unmutedBy
			count++
		}
	}
	return count, nil
}

func (m *mockMuteRepo) ListByOffering(ctx context.Context, offeringID uuid.UUID, params pagination.PageParams, filters MuteFilters) ([]MuteWithUser, bool, error) {
	return nil, false, nil
}

func (m *mockMuteRepo) ListAll(ctx context.Context, params pagination.PageParams, filters MuteFilters) ([]MuteWithUser, bool, error) {
	return nil, false, nil
}

type mockOfferingChecker struct {
	offerings map[uuid.UUID]bool
}

func newMockOfferingChecker() *mockOfferingChecker {
	return &mockOfferingChecker{offerings: make(map[uuid.UUID]bool)}
}

func (m *mockOfferingChecker) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	return m.offerings[id], nil
}

type mockUserChecker struct {
	users map[uuid.UUID]bool
}

func newMockUserChecker() *mockUserChecker {
	return &mockUserChecker{users: make(map[uuid.UUID]bool)}
}

func (m *mockUserChecker) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	return m.users[id], nil
}

// Tests

func TestMuteInCourse(t *testing.T) {
	muteRepo := newMockMuteRepo()
	offeringChecker := newMockOfferingChecker()
	userChecker := newMockUserChecker()
	service := NewService(muteRepo, offeringChecker, userChecker)

	userID := uuid.New()
	offeringID := uuid.New()
	mutedBy := uuid.New()

	userChecker.users[userID] = true
	offeringChecker.offerings[offeringID] = true

	reason := "test reason"
	mute, err := service.MuteInCourse(context.Background(), userID, offeringID, mutedBy, &reason, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mute.UserID != userID {
		t.Errorf("UserID = %v, want %v", mute.UserID, userID)
	}
	if mute.ScopeType != ScopeCourse {
		t.Errorf("ScopeType = %v, want %v", mute.ScopeType, ScopeCourse)
	}
}

func TestMuteInCourse_SelfMute(t *testing.T) {
	muteRepo := newMockMuteRepo()
	offeringChecker := newMockOfferingChecker()
	userChecker := newMockUserChecker()
	service := NewService(muteRepo, offeringChecker, userChecker)

	userID := uuid.New()
	offeringID := uuid.New()

	_, err := service.MuteInCourse(context.Background(), userID, offeringID, userID, nil, nil)

	if err != ErrCannotMuteSelf {
		t.Errorf("err = %v, want ErrCannotMuteSelf", err)
	}
}

func TestMuteInCourse_AlreadyMuted(t *testing.T) {
	muteRepo := newMockMuteRepo()
	offeringChecker := newMockOfferingChecker()
	userChecker := newMockUserChecker()
	service := NewService(muteRepo, offeringChecker, userChecker)

	userID := uuid.New()
	offeringID := uuid.New()
	mutedBy := uuid.New()

	userChecker.users[userID] = true
	offeringChecker.offerings[offeringID] = true

	// First mute
	_, _ = service.MuteInCourse(context.Background(), userID, offeringID, mutedBy, nil, nil)

	// Second mute should fail
	_, err := service.MuteInCourse(context.Background(), userID, offeringID, mutedBy, nil, nil)

	if err != ErrAlreadyMuted {
		t.Errorf("err = %v, want ErrAlreadyMuted", err)
	}
}

func TestMuteInCourse_UserNotFound(t *testing.T) {
	muteRepo := newMockMuteRepo()
	offeringChecker := newMockOfferingChecker()
	userChecker := newMockUserChecker()
	service := NewService(muteRepo, offeringChecker, userChecker)

	userID := uuid.New()
	offeringID := uuid.New()
	mutedBy := uuid.New()

	// User does not exist
	offeringChecker.offerings[offeringID] = true

	_, err := service.MuteInCourse(context.Background(), userID, offeringID, mutedBy, nil, nil)

	if err != ErrUserNotFound {
		t.Errorf("err = %v, want ErrUserNotFound", err)
	}
}

func TestMuteInCourse_OfferingNotFound(t *testing.T) {
	muteRepo := newMockMuteRepo()
	offeringChecker := newMockOfferingChecker()
	userChecker := newMockUserChecker()
	service := NewService(muteRepo, offeringChecker, userChecker)

	userID := uuid.New()
	offeringID := uuid.New()
	mutedBy := uuid.New()

	userChecker.users[userID] = true
	// Offering does not exist

	_, err := service.MuteInCourse(context.Background(), userID, offeringID, mutedBy, nil, nil)

	if err != ErrOfferingNotFound {
		t.Errorf("err = %v, want ErrOfferingNotFound", err)
	}
}

func TestMuteUniversityWide(t *testing.T) {
	muteRepo := newMockMuteRepo()
	offeringChecker := newMockOfferingChecker()
	userChecker := newMockUserChecker()
	service := NewService(muteRepo, offeringChecker, userChecker)

	userID := uuid.New()
	mutedBy := uuid.New()

	userChecker.users[userID] = true

	mute, err := service.MuteUniversityWide(context.Background(), userID, mutedBy, nil, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mute.ScopeType != ScopeUniversity {
		t.Errorf("ScopeType = %v, want %v", mute.ScopeType, ScopeUniversity)
	}
	if mute.ScopeID != nil {
		t.Error("ScopeID should be nil for university-wide mute")
	}
}

func TestUnmute(t *testing.T) {
	muteRepo := newMockMuteRepo()
	offeringChecker := newMockOfferingChecker()
	userChecker := newMockUserChecker()
	service := NewService(muteRepo, offeringChecker, userChecker)

	userID := uuid.New()
	offeringID := uuid.New()
	mutedBy := uuid.New()
	unmutedBy := uuid.New()

	userChecker.users[userID] = true
	offeringChecker.offerings[offeringID] = true

	mute, _ := service.MuteInCourse(context.Background(), userID, offeringID, mutedBy, nil, nil)

	err := service.Unmute(context.Background(), mute.ID, unmutedBy)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify unmuted
	updated, _ := muteRepo.GetByID(context.Background(), mute.ID)
	if updated.UnmutedAt == nil {
		t.Error("UnmutedAt should be set")
	}
	if updated.UnmutedBy == nil || *updated.UnmutedBy != unmutedBy {
		t.Error("UnmutedBy should be set")
	}
}

func TestUnmuteAll(t *testing.T) {
	muteRepo := newMockMuteRepo()
	offeringChecker := newMockOfferingChecker()
	userChecker := newMockUserChecker()
	service := NewService(muteRepo, offeringChecker, userChecker)

	userID := uuid.New()
	mutedBy := uuid.New()
	unmutedBy := uuid.New()
	offeringA := uuid.New()
	offeringB := uuid.New()

	userChecker.users[userID] = true
	offeringChecker.offerings[offeringA] = true
	offeringChecker.offerings[offeringB] = true

	// Create two mutes
	_, _ = service.MuteInCourse(context.Background(), userID, offeringA, mutedBy, nil, nil)
	_, _ = service.MuteInCourse(context.Background(), userID, offeringB, mutedBy, nil, nil)

	count, err := service.UnmuteAll(context.Background(), userID, unmutedBy)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

func TestIsMuted(t *testing.T) {
	muteRepo := newMockMuteRepo()
	offeringChecker := newMockOfferingChecker()
	userChecker := newMockUserChecker()
	service := NewService(muteRepo, offeringChecker, userChecker)

	userID := uuid.New()
	offeringID := uuid.New()
	mutedBy := uuid.New()

	userChecker.users[userID] = true
	offeringChecker.offerings[offeringID] = true

	// Not muted initially
	muted, _ := service.IsMuted(context.Background(), userID, &offeringID)
	if muted {
		t.Error("should not be muted initially")
	}

	// Mute in course
	_, _ = service.MuteInCourse(context.Background(), userID, offeringID, mutedBy, nil, nil)

	muted, _ = service.IsMuted(context.Background(), userID, &offeringID)
	if !muted {
		t.Error("should be muted after MuteInCourse")
	}

	// Different offering should not be muted
	otherOffering := uuid.New()
	muted, _ = service.IsMuted(context.Background(), userID, &otherOffering)
	if muted {
		t.Error("should not be muted in different offering")
	}
}

func TestIsMuted_UniversityWide(t *testing.T) {
	muteRepo := newMockMuteRepo()
	offeringChecker := newMockOfferingChecker()
	userChecker := newMockUserChecker()
	service := NewService(muteRepo, offeringChecker, userChecker)

	userID := uuid.New()
	mutedBy := uuid.New()
	offeringID := uuid.New()

	userChecker.users[userID] = true

	// Mute university-wide
	_, _ = service.MuteUniversityWide(context.Background(), userID, mutedBy, nil, nil)

	// Should be muted in any offering
	muted, _ := service.IsMuted(context.Background(), userID, &offeringID)
	if !muted {
		t.Error("should be muted university-wide")
	}
}
