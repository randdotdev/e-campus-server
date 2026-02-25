package subscription

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

// mockRepository for testing
type mockRepository struct {
	subscription *Subscription
	tierLimits   map[string]*TierLimits
	history      []History
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		subscription: &Subscription{
			ID:   uuid.New(),
			Tier: TierBasic,
		},
		tierLimits: map[string]*TierLimits{
			TierFree: {
				Tier:                     TierFree,
				MaxColleges:              5,
				MaxDepartmentsPerCollege: 10,
				MaxProgramsPerDepartment: 5,
				MaxStudentsPerProgram:    100,
				MaxApplicationsPerUser:   3,
				MaxStaffUsers:            20,
				MaxStorageBytes:          5368709120,
				MaxFileSizeBytes:         104857600,
			},
			TierBasic: {
				Tier:                     TierBasic,
				MaxColleges:              10,
				MaxDepartmentsPerCollege: 20,
				MaxProgramsPerDepartment: 15,
				MaxStudentsPerProgram:    300,
				MaxApplicationsPerUser:   5,
				MaxStaffUsers:            100,
				MaxStorageBytes:          53687091200,
				MaxFileSizeBytes:         209715200,
			},
			TierPremium: {
				Tier:                     TierPremium,
				MaxColleges:              100,
				MaxDepartmentsPerCollege: 50,
				MaxProgramsPerDepartment: 30,
				MaxStudentsPerProgram:    1000,
				MaxApplicationsPerUser:   10,
				MaxStaffUsers:            500,
				MaxStorageBytes:          536870912000,
				MaxFileSizeBytes:         1073741824,
			},
		},
		history: []History{},
	}
}

func (m *mockRepository) Get(ctx context.Context) (*Subscription, error) {
	return m.subscription, nil
}

func (m *mockRepository) Update(ctx context.Context, sub *Subscription) error {
	m.subscription = sub
	return nil
}

func (m *mockRepository) GetTierLimits(ctx context.Context, tier string) (*TierLimits, error) {
	tl, ok := m.tierLimits[tier]
	if !ok {
		return nil, ErrInvalidTier
	}
	return tl, nil
}

func (m *mockRepository) GetAllTierLimits(ctx context.Context) ([]TierLimits, error) {
	var result []TierLimits
	for _, tl := range m.tierLimits {
		result = append(result, *tl)
	}
	return result, nil
}

func (m *mockRepository) UpdateTierLimits(ctx context.Context, tl *TierLimits) error {
	m.tierLimits[tl.Tier] = tl
	return nil
}

func (m *mockRepository) AddHistory(ctx context.Context, sub *Subscription, reason string, changedBy *uuid.UUID) error {
	m.history = append(m.history, History{
		ID:           uuid.New(),
		Tier:         sub.Tier,
		ChangedBy:    changedBy,
		ChangedAt:    time.Now(),
		ChangeReason: &reason,
	})
	return nil
}

func (m *mockRepository) GetHistory(ctx context.Context, limit int) ([]History, error) {
	if limit > len(m.history) {
		limit = len(m.history)
	}
	return m.history[:limit], nil
}

func (m *mockRepository) UpdateWithHistory(ctx context.Context, sub *Subscription, reason string, changedBy *uuid.UUID) error {
	m.subscription = sub
	return m.AddHistory(ctx, sub, reason, changedBy)
}

func TestServiceGetLimits(t *testing.T) {
	t.Run("returns limits for current tier", func(t *testing.T) {
		repo := newMockRepository()
		service := NewService(repo)

		limits, err := service.GetLimits(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if limits.MaxColleges != 10 {
			t.Errorf("MaxColleges = %d, want 10", limits.MaxColleges)
		}
		if limits.MaxStorageBytes != 53687091200 {
			t.Errorf("MaxStorageBytes = %d, want 53687091200", limits.MaxStorageBytes)
		}
	})

	t.Run("applies overrides", func(t *testing.T) {
		repo := newMockRepository()
		override := 50
		repo.subscription.MaxCollegesOverride = &override
		service := NewService(repo)

		limits, err := service.GetLimits(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if limits.MaxColleges != 50 {
			t.Errorf("MaxColleges = %d, want 50 (overridden)", limits.MaxColleges)
		}
	})

	t.Run("falls back to free tier when expired", func(t *testing.T) {
		repo := newMockRepository()
		expired := time.Now().Add(-time.Hour)
		repo.subscription.ExpiresAt = &expired
		service := NewService(repo)

		limits, err := service.GetLimits(context.Background())
		if err != ErrSubscriptionExpired {
			t.Errorf("err = %v, want ErrSubscriptionExpired", err)
		}

		if limits.MaxColleges != 5 {
			t.Errorf("MaxColleges = %d, want 5 (free tier)", limits.MaxColleges)
		}
	})
}

func TestServiceUpdateTier(t *testing.T) {
	t.Run("updates tier successfully", func(t *testing.T) {
		repo := newMockRepository()
		service := NewService(repo)
		userID := uuid.New()

		sub, err := service.UpdateTier(context.Background(), TierPremium, "upgrade", userID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if sub.Tier != TierPremium {
			t.Errorf("Tier = %v, want %v", sub.Tier, TierPremium)
		}

		if len(repo.history) != 1 {
			t.Errorf("history len = %d, want 1", len(repo.history))
		}
	})

	t.Run("rejects invalid tier", func(t *testing.T) {
		repo := newMockRepository()
		service := NewService(repo)
		userID := uuid.New()

		_, err := service.UpdateTier(context.Background(), "enterprise", "upgrade", userID)
		if err != ErrInvalidTier {
			t.Errorf("err = %v, want ErrInvalidTier", err)
		}
	})
}

func TestServiceSetOverrides(t *testing.T) {
	t.Run("sets overrides", func(t *testing.T) {
		repo := newMockRepository()
		service := NewService(repo)
		userID := uuid.New()

		colleges := 100
		storage := int64(107374182400)
		overrides := Overrides{
			MaxColleges: &colleges,
			MaxStorage:  &storage,
		}

		sub, err := service.SetOverrides(context.Background(), overrides, "custom limits", userID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if sub.MaxCollegesOverride == nil || *sub.MaxCollegesOverride != 100 {
			t.Errorf("MaxCollegesOverride = %v, want 100", sub.MaxCollegesOverride)
		}
		if sub.MaxStorageOverride == nil || *sub.MaxStorageOverride != storage {
			t.Errorf("MaxStorageOverride = %v, want %d", sub.MaxStorageOverride, storage)
		}
	})
}

func TestServiceClearOverrides(t *testing.T) {
	t.Run("clears all overrides", func(t *testing.T) {
		repo := newMockRepository()
		colleges := 100
		storage := int64(107374182400)
		repo.subscription.MaxCollegesOverride = &colleges
		repo.subscription.MaxStorageOverride = &storage
		service := NewService(repo)
		userID := uuid.New()

		sub, err := service.ClearOverrides(context.Background(), "reset", userID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if sub.MaxCollegesOverride != nil {
			t.Error("MaxCollegesOverride should be nil")
		}
		if sub.MaxStorageOverride != nil {
			t.Error("MaxStorageOverride should be nil")
		}
	})
}

func TestServiceUpdateTierLimits(t *testing.T) {
	t.Run("updates tier limits", func(t *testing.T) {
		repo := newMockRepository()
		service := NewService(repo)

		tl := &TierLimits{
			Tier:            TierBasic,
			MaxColleges:     20,
			MaxStorageBytes: 107374182400,
		}

		err := service.UpdateTierLimits(context.Background(), tl)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		updated := repo.tierLimits[TierBasic]
		if updated.MaxColleges != 20 {
			t.Errorf("MaxColleges = %d, want 20", updated.MaxColleges)
		}
	})

	t.Run("rejects invalid tier", func(t *testing.T) {
		repo := newMockRepository()
		service := NewService(repo)

		tl := &TierLimits{Tier: "enterprise"}
		err := service.UpdateTierLimits(context.Background(), tl)
		if err != ErrInvalidTier {
			t.Errorf("err = %v, want ErrInvalidTier", err)
		}
	})
}

func TestServiceGetHistory(t *testing.T) {
	t.Run("returns history with default limit", func(t *testing.T) {
		repo := newMockRepository()
		service := NewService(repo)

		history, err := service.GetHistory(context.Background(), 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if history == nil {
			t.Error("history should not be nil")
		}
	})
}
