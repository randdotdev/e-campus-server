package subscription

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestToSubscriptionResponse(t *testing.T) {
	now := time.Now()
	subID := uuid.New()
	userID := uuid.New()

	t.Run("without overrides", func(t *testing.T) {
		sub := &Subscription{
			ID:        subID,
			Tier:      TierBasic,
			UpdatedBy: &userID,
			UpdatedAt: now,
		}
		limits := Limits{
			MaxColleges:      10,
			MaxStorageBytes:  53687091200,
			MaxFileSizeBytes: 209715200,
		}

		resp := ToSubscriptionResponse(sub, limits)

		if resp.ID != subID {
			t.Errorf("ID = %v, want %v", resp.ID, subID)
		}
		if resp.Tier != TierBasic {
			t.Errorf("Tier = %v, want %v", resp.Tier, TierBasic)
		}
		if resp.Limits.MaxStorageBytes != 53687091200 {
			t.Errorf("Limits.MaxStorageBytes = %v, want 53687091200", resp.Limits.MaxStorageBytes)
		}
		if resp.Overrides != nil {
			t.Error("Overrides should be nil when no overrides set")
		}
	})

	t.Run("with storage overrides", func(t *testing.T) {
		storageOverride := int64(107374182400)
		fileSizeOverride := int64(524288000)
		sub := &Subscription{
			ID:                  subID,
			Tier:                TierBasic,
			MaxStorageOverride:  &storageOverride,
			MaxFileSizeOverride: &fileSizeOverride,
			UpdatedAt:           now,
		}
		limits := Limits{
			MaxColleges:      10,
			MaxStorageBytes:  107374182400,
			MaxFileSizeBytes: 524288000,
		}

		resp := ToSubscriptionResponse(sub, limits)

		if resp.Overrides == nil {
			t.Fatal("Overrides should not be nil")
		}
		if *resp.Overrides.MaxStorage != storageOverride {
			t.Errorf("Overrides.MaxStorage = %v, want %v", *resp.Overrides.MaxStorage, storageOverride)
		}
		if *resp.Overrides.MaxFileSize != fileSizeOverride {
			t.Errorf("Overrides.MaxFileSize = %v, want %v", *resp.Overrides.MaxFileSize, fileSizeOverride)
		}
	})
}

func TestToTierLimitsResponse(t *testing.T) {
	now := time.Now()
	tl := &TierLimits{
		Tier:                     TierPremium,
		MaxColleges:              100,
		MaxDepartmentsPerCollege: 50,
		MaxProgramsPerDepartment: 30,
		MaxStudentsPerProgram:    1000,
		MaxApplicationsPerUser:   10,
		MaxStaffUsers:            500,
		MaxStorageBytes:          536870912000,
		MaxFileSizeBytes:         1073741824,
		UpdatedAt:                now,
	}

	resp := ToTierLimitsResponse(tl)

	if resp.Tier != TierPremium {
		t.Errorf("Tier = %v, want %v", resp.Tier, TierPremium)
	}
	if resp.MaxColleges != 100 {
		t.Errorf("MaxColleges = %v, want 100", resp.MaxColleges)
	}
	if resp.MaxStorageBytes != 536870912000 {
		t.Errorf("MaxStorageBytes = %v, want 536870912000", resp.MaxStorageBytes)
	}
	if resp.MaxFileSizeBytes != 1073741824 {
		t.Errorf("MaxFileSizeBytes = %v, want 1073741824", resp.MaxFileSizeBytes)
	}
}

func TestToTierLimitsResponses(t *testing.T) {
	tiers := []TierLimits{
		{Tier: TierFree, MaxStorageBytes: 5368709120},
		{Tier: TierBasic, MaxStorageBytes: 53687091200},
		{Tier: TierPremium, MaxStorageBytes: 536870912000},
	}

	resp := ToTierLimitsResponses(tiers)

	if len(resp) != 3 {
		t.Fatalf("len = %d, want 3", len(resp))
	}
	if resp[0].Tier != TierFree {
		t.Errorf("resp[0].Tier = %v, want %v", resp[0].Tier, TierFree)
	}
	if resp[2].MaxStorageBytes != 536870912000 {
		t.Errorf("resp[2].MaxStorageBytes = %v, want 536870912000", resp[2].MaxStorageBytes)
	}
}

func TestToHistoryResponse(t *testing.T) {
	now := time.Now()
	histID := uuid.New()
	userID := uuid.New()
	reason := "upgraded plan"

	t.Run("without overrides", func(t *testing.T) {
		h := &History{
			ID:           histID,
			Tier:         TierBasic,
			ChangedBy:    &userID,
			ChangedAt:    now,
			ChangeReason: &reason,
		}

		resp := ToHistoryResponse(h)

		if resp.ID != histID {
			t.Errorf("ID = %v, want %v", resp.ID, histID)
		}
		if resp.Tier != TierBasic {
			t.Errorf("Tier = %v, want %v", resp.Tier, TierBasic)
		}
		if resp.Overrides != nil {
			t.Error("Overrides should be nil when no overrides set")
		}
	})

	t.Run("with storage overrides", func(t *testing.T) {
		storageOverride := int64(107374182400)
		h := &History{
			ID:                 histID,
			Tier:               TierBasic,
			MaxStorageOverride: &storageOverride,
			ChangedAt:          now,
		}

		resp := ToHistoryResponse(h)

		if resp.Overrides == nil {
			t.Fatal("Overrides should not be nil")
		}
		if *resp.Overrides.MaxStorage != storageOverride {
			t.Errorf("Overrides.MaxStorage = %v, want %v", *resp.Overrides.MaxStorage, storageOverride)
		}
	})
}

func TestSetOverridesRequestToOverrides(t *testing.T) {
	storage := int64(107374182400)
	fileSize := int64(524288000)
	colleges := 50

	req := SetOverridesRequest{
		MaxColleges: &colleges,
		MaxStorage:  &storage,
		MaxFileSize: &fileSize,
		Reason:      "custom limits",
	}

	overrides := req.ToOverrides()

	if *overrides.MaxColleges != colleges {
		t.Errorf("MaxColleges = %v, want %v", *overrides.MaxColleges, colleges)
	}
	if *overrides.MaxStorage != storage {
		t.Errorf("MaxStorage = %v, want %v", *overrides.MaxStorage, storage)
	}
	if *overrides.MaxFileSize != fileSize {
		t.Errorf("MaxFileSize = %v, want %v", *overrides.MaxFileSize, fileSize)
	}
}
