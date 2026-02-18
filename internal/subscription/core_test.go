package subscription

import (
	"testing"
	"time"
)

func TestToLimits(t *testing.T) {
	tl := &TierLimits{
		Tier:                     TierBasic,
		MaxColleges:              10,
		MaxDepartmentsPerCollege: 20,
		MaxProgramsPerDepartment: 15,
		MaxStudentsPerProgram:    300,
		MaxApplicationsPerUser:   5,
		MaxStaffUsers:            100,
	}

	limits := ToLimits(tl)

	if limits.MaxColleges != 10 {
		t.Errorf("MaxColleges = %d, want 10", limits.MaxColleges)
	}
	if limits.MaxDepartmentsPerCollege != 20 {
		t.Errorf("MaxDepartmentsPerCollege = %d, want 20", limits.MaxDepartmentsPerCollege)
	}
	if limits.MaxStaffUsers != 100 {
		t.Errorf("MaxStaffUsers = %d, want 100", limits.MaxStaffUsers)
	}
}

func TestApplyOverrides(t *testing.T) {
	base := Limits{
		MaxColleges:              10,
		MaxDepartmentsPerCollege: 20,
		MaxProgramsPerDepartment: 15,
		MaxStudentsPerProgram:    300,
		MaxApplicationsPerUser:   5,
		MaxStaffUsers:            100,
	}

	t.Run("no overrides", func(t *testing.T) {
		sub := &Subscription{}
		result := ApplyOverrides(base, sub)

		if result.MaxColleges != 10 {
			t.Errorf("MaxColleges = %d, want 10", result.MaxColleges)
		}
	})

	t.Run("partial override", func(t *testing.T) {
		override := 50
		sub := &Subscription{
			MaxCollegesOverride: &override,
		}
		result := ApplyOverrides(base, sub)

		if result.MaxColleges != 50 {
			t.Errorf("MaxColleges = %d, want 50 (overridden)", result.MaxColleges)
		}
		if result.MaxDepartmentsPerCollege != 20 {
			t.Errorf("MaxDepartmentsPerCollege = %d, want 20 (unchanged)", result.MaxDepartmentsPerCollege)
		}
	})

	t.Run("all overrides", func(t *testing.T) {
		c, d, p, s, a, st := 100, 50, 30, 500, 10, 200
		sub := &Subscription{
			MaxCollegesOverride:     &c,
			MaxDepartmentsOverride:  &d,
			MaxProgramsOverride:     &p,
			MaxStudentsOverride:     &s,
			MaxApplicationsOverride: &a,
			MaxStaffOverride:        &st,
		}
		result := ApplyOverrides(base, sub)

		if result.MaxColleges != 100 {
			t.Errorf("MaxColleges = %d, want 100", result.MaxColleges)
		}
		if result.MaxStaffUsers != 200 {
			t.Errorf("MaxStaffUsers = %d, want 200", result.MaxStaffUsers)
		}
	})
}

func TestHasOverrides(t *testing.T) {
	t.Run("no overrides", func(t *testing.T) {
		sub := &Subscription{}
		if HasOverrides(sub) {
			t.Error("HasOverrides() = true, want false")
		}
	})

	t.Run("has override", func(t *testing.T) {
		override := 50
		sub := &Subscription{MaxCollegesOverride: &override}
		if !HasOverrides(sub) {
			t.Error("HasOverrides() = false, want true")
		}
	})
}

func TestIsExpired(t *testing.T) {
	past := time.Now().Add(-time.Hour)
	future := time.Now().Add(time.Hour)

	tests := []struct {
		name      string
		expiresAt *time.Time
		want      bool
	}{
		{"nil never expires", nil, false},
		{"past is expired", &past, true},
		{"future not expired", &future, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsExpired(tt.expiresAt); got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidTier(t *testing.T) {
	tests := []struct {
		tier string
		want bool
	}{
		{TierFree, true},
		{TierBasic, true},
		{TierPremium, true},
		{"enterprise", false},
		{"", false},
		{"FREE", false},
	}

	for _, tt := range tests {
		t.Run(tt.tier, func(t *testing.T) {
			if got := IsValidTier(tt.tier); got != tt.want {
				t.Errorf("IsValidTier(%q) = %v, want %v", tt.tier, got, tt.want)
			}
		})
	}
}

func TestCanCreate(t *testing.T) {
	tests := []struct {
		name    string
		current int
		limit   int
		want    bool
	}{
		{"zero of limit", 0, 5, true},
		{"below limit", 3, 5, true},
		{"at limit", 5, 5, false},
		{"over limit", 7, 5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CanCreate(tt.current, tt.limit); got != tt.want {
				t.Errorf("CanCreate(%d, %d) = %v, want %v", tt.current, tt.limit, got, tt.want)
			}
		})
	}
}

func TestRemaining(t *testing.T) {
	tests := []struct {
		name    string
		current int
		limit   int
		want    int
	}{
		{"none used", 0, 5, 5},
		{"some used", 3, 5, 2},
		{"all used", 5, 5, 0},
		{"over limit", 7, 5, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Remaining(tt.current, tt.limit); got != tt.want {
				t.Errorf("Remaining(%d, %d) = %d, want %d", tt.current, tt.limit, got, tt.want)
			}
		})
	}
}
