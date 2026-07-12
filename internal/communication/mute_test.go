package communication

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/shared/pagination"
)

type muteMockRepo struct {
	mutes map[uuid.UUID]*Mute
}

// Create mirrors the partial unique index: a second open mute for the same
// user and scope is ErrAlreadyMuted.
func (m *muteMockRepo) Create(ctx context.Context, mt *Mute) error {
	for _, e := range m.mutes {
		if e.UnmutedAt == nil && e.UserID == mt.UserID && e.ScopeType == mt.ScopeType && eqID(e.ScopeID, mt.ScopeID) {
			return ErrAlreadyMuted
		}
	}
	m.mutes[mt.ID] = mt
	return nil
}
func (m *muteMockRepo) GetByID(ctx context.Context, id uuid.UUID) (*Mute, error) {
	return m.mutes[id], nil
}
func (m *muteMockRepo) IsMuted(ctx context.Context, userID uuid.UUID, offeringID *uuid.UUID) (bool, error) {
	return false, nil
}
func (m *muteMockRepo) Unmute(ctx context.Context, id, by uuid.UUID) error { return nil }
func (m *muteMockRepo) UnmuteAll(ctx context.Context, userID, by uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *muteMockRepo) ListByOffering(ctx context.Context, o uuid.UUID, p pagination.PageParams, f MuteFilters) ([]MuteWithUser, bool, error) {
	return nil, false, nil
}
func (m *muteMockRepo) ListAll(ctx context.Context, p pagination.PageParams, f MuteFilters) ([]MuteWithUser, bool, error) {
	return nil, false, nil
}

func eqID(a, b *uuid.UUID) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

type alwaysExists struct{}

func (alwaysExists) Exists(ctx context.Context, id uuid.UUID) (bool, error) { return true, nil }

type neverExists struct{}

func (neverExists) Exists(ctx context.Context, id uuid.UUID) (bool, error) { return false, nil }

func TestMuteActive(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	if (&Mute{}).Active(now) != true {
		t.Error("fresh mute should be active")
	}
	if (&Mute{UnmutedAt: &past}).Active(now) {
		t.Error("unmuted should be inactive")
	}
	if (&Mute{ExpiresAt: &past}).Active(now) {
		t.Error("expired should be inactive")
	}
	if !(&Mute{ExpiresAt: &future}).Active(now) {
		t.Error("not-yet-expired should be active")
	}
}

func TestMuteInOfferingRules(t *testing.T) {
	ctx := context.Background()
	repo := &muteMockRepo{mutes: map[uuid.UUID]*Mute{}}
	s := NewMuteService(repo, alwaysExists{}, alwaysExists{})
	actor := uuid.New()
	user := uuid.New()
	offering := uuid.New()

	// cannot mute self
	if _, err := s.MuteInOffering(ctx, actor, offering, actor, nil, nil); err != ErrCannotMuteSelf {
		t.Errorf("self-mute = %v, want ErrCannotMuteSelf", err)
	}
	// happy path
	mute, err := s.MuteInOffering(ctx, user, offering, actor, nil, nil)
	if err != nil {
		t.Fatalf("mute = %v", err)
	}
	if mute.ScopeType != ScopeOffering {
		t.Errorf("scope = %v, want offering", mute.ScopeType)
	}
	// already muted: a second open mute for the same user + offering (the
	// partial unique index, surfaced by the mock).
	if _, err := s.MuteInOffering(ctx, user, offering, actor, nil, nil); err != ErrAlreadyMuted {
		t.Errorf("re-mute = %v, want ErrAlreadyMuted", err)
	}
}

func TestMuteInOfferingMissingUser(t *testing.T) {
	ctx := context.Background()
	repo := &muteMockRepo{mutes: map[uuid.UUID]*Mute{}}
	s := NewMuteService(repo, alwaysExists{}, neverExists{})
	if _, err := s.MuteInOffering(ctx, uuid.New(), uuid.New(), uuid.New(), nil, nil); err != ErrUserNotFound {
		t.Errorf("missing user = %v, want ErrUserNotFound", err)
	}
}
