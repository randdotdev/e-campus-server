package subscription

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestValidTier(t *testing.T) {
	for _, tt := range []struct {
		tier Tier
		want bool
	}{
		{TierFree, true}, {TierBasic, true}, {TierPremium, true},
		{"enterprise", false}, {"", false},
	} {
		if got := ValidTier(tt.tier); got != tt.want {
			t.Errorf("ValidTier(%q) = %v, want %v", tt.tier, got, tt.want)
		}
	}
}

func TestIsExpired(t *testing.T) {
	past := time.Now().Add(-time.Hour)
	future := time.Now().Add(time.Hour)
	if !IsExpired(&past) {
		t.Error("past should be expired")
	}
	if IsExpired(&future) {
		t.Error("future should not be expired")
	}
	if IsExpired(nil) {
		t.Error("nil should never expire")
	}
}

func TestCanCreateAndRemaining(t *testing.T) {
	if !CanCreate(2, 3) {
		t.Error("2 < 3 should allow create")
	}
	if CanCreate(3, 3) {
		t.Error("3 >= 3 should block create")
	}
	if got := Remaining(1, 3); got != 2 {
		t.Errorf("Remaining(1,3) = %d, want 2", got)
	}
	if got := Remaining(5, 3); got != 0 {
		t.Errorf("Remaining(5,3) = %d, want 0", got)
	}
}

func TestApplyOverrides(t *testing.T) {
	base := Limits{MaxColleges: 1, MaxStaffUsers: 10}
	ten := 10
	sub := &Subscription{MaxCollegesOverride: &ten}
	got := ApplyOverrides(base, sub)
	if got.MaxColleges != 10 {
		t.Errorf("MaxColleges = %d, want 10 (override)", got.MaxColleges)
	}
	if got.MaxStaffUsers != 10 {
		t.Errorf("MaxStaffUsers = %d, want 10 (base)", got.MaxStaffUsers)
	}
}

// mockRepo is an in-memory subscription.Repository.
type mockRepo struct {
	sub   *Subscription
	tiers map[Tier]*TierLimits
	// forceConflict makes every UpdateWithHistory lose the version CAS.
	forceConflict bool
}

func (m *mockRepo) Get(ctx context.Context) (*Subscription, error) { return m.sub, nil }
func (m *mockRepo) GetTierLimits(ctx context.Context, tier Tier) (*TierLimits, error) {
	tl := m.tiers[tier]
	if tl == nil {
		return nil, ErrTierNotFound
	}
	return tl, nil
}
func (m *mockRepo) GetAllTierLimits(ctx context.Context) ([]TierLimits, error) { return nil, nil }
func (m *mockRepo) UpdateTierLimits(ctx context.Context, tl *TierLimits) error { return nil }
func (m *mockRepo) GetHistory(ctx context.Context, limit int) ([]History, error) {
	return nil, nil
}
func (m *mockRepo) UpdateWithHistory(ctx context.Context, sub *Subscription, expectedVersion int64, reason string, changedBy *uuid.UUID) (int64, error) {
	if m.forceConflict {
		return 0, ErrConflict
	}
	if m.sub != nil && m.sub.Version != expectedVersion {
		return 0, ErrConflict
	}
	sub.Version = expectedVersion + 1
	m.sub = sub
	return sub.Version, nil
}

func newTestService(sub *Subscription) (*Service, *mockRepo) {
	repo := &mockRepo{
		sub: sub,
		tiers: map[Tier]*TierLimits{
			TierFree:    {Tier: TierFree, MaxColleges: 1},
			TierPremium: {Tier: TierPremium, MaxColleges: 100},
		},
	}
	return NewService(repo), repo
}

func TestServiceGetLimitsAppliesTierAndOverrides(t *testing.T) {
	fifty := 50
	s, _ := newTestService(&Subscription{Tier: TierPremium, MaxCollegesOverride: &fifty})
	limits, err := s.GetLimits(context.Background())
	if err != nil {
		t.Fatalf("GetLimits = %v", err)
	}
	if limits.MaxColleges != 50 {
		t.Errorf("MaxColleges = %d, want 50 (override on premium)", limits.MaxColleges)
	}
}

func TestServiceGetLimitsExpiredFallsBackToFree(t *testing.T) {
	past := time.Now().Add(-time.Hour)
	s, _ := newTestService(&Subscription{Tier: TierPremium, ExpiresAt: &past})
	limits, err := s.GetLimits(context.Background())
	if err != ErrSubscriptionExpired {
		t.Errorf("err = %v, want ErrSubscriptionExpired", err)
	}
	if limits.MaxColleges != 1 {
		t.Errorf("MaxColleges = %d, want 1 (free fallback)", limits.MaxColleges)
	}
}

func TestServiceUpdateTierValidation(t *testing.T) {
	s, _ := newTestService(&Subscription{Tier: TierFree})
	ctx := context.Background()
	if _, err := s.UpdateTier(ctx, "enterprise", "x", uuid.New()); err != ErrInvalidTier {
		t.Errorf("invalid tier = %v, want ErrInvalidTier", err)
	}
	sub, err := s.UpdateTier(ctx, TierPremium, "upgrade", uuid.New())
	if err != nil {
		t.Fatalf("valid tier = %v", err)
	}
	if sub.Tier != TierPremium {
		t.Errorf("tier = %v, want premium", sub.Tier)
	}
}

func TestServiceUpdateTierBumpsVersion(t *testing.T) {
	s, _ := newTestService(&Subscription{Tier: TierFree, Version: 4})
	sub, err := s.UpdateTier(context.Background(), TierPremium, "upgrade", uuid.New())
	if err != nil {
		t.Fatalf("update = %v", err)
	}
	if sub.Version != 5 {
		t.Errorf("version = %d, want 5", sub.Version)
	}
}

func TestServiceSetOverridesPersistsStorage(t *testing.T) {
	s, repo := newTestService(&Subscription{Tier: TierFree})
	storage := int64(999)
	if _, err := s.SetOverrides(context.Background(), Overrides{MaxStorage: &storage}, "bump", uuid.New()); err != nil {
		t.Fatalf("set overrides = %v", err)
	}
	if repo.sub.MaxStorageOverride == nil || *repo.sub.MaxStorageOverride != storage {
		t.Errorf("storage override = %v, want %d", repo.sub.MaxStorageOverride, storage)
	}
}

func TestServiceUpdateTierConflict(t *testing.T) {
	s, repo := newTestService(&Subscription{Tier: TierFree})
	repo.forceConflict = true
	if _, err := s.UpdateTier(context.Background(), TierPremium, "upgrade", uuid.New()); err != ErrConflict {
		t.Errorf("update under permanent conflict = %v, want ErrConflict", err)
	}
}
