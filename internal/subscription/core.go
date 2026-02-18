package subscription

import "time"

// Limits represents effective limits for the system.
type Limits struct {
	MaxColleges              int
	MaxDepartmentsPerCollege int
	MaxProgramsPerDepartment int
	MaxStudentsPerProgram    int
	MaxApplicationsPerUser   int
	MaxStaffUsers            int
}

// ToLimits converts TierLimits to Limits.
func ToLimits(tl *TierLimits) Limits {
	return Limits{
		MaxColleges:              tl.MaxColleges,
		MaxDepartmentsPerCollege: tl.MaxDepartmentsPerCollege,
		MaxProgramsPerDepartment: tl.MaxProgramsPerDepartment,
		MaxStudentsPerProgram:    tl.MaxStudentsPerProgram,
		MaxApplicationsPerUser:   tl.MaxApplicationsPerUser,
		MaxStaffUsers:            tl.MaxStaffUsers,
	}
}

// ApplyOverrides applies subscription overrides to base limits.
func ApplyOverrides(base Limits, sub *Subscription) Limits {
	if sub.MaxCollegesOverride != nil {
		base.MaxColleges = *sub.MaxCollegesOverride
	}
	if sub.MaxDepartmentsOverride != nil {
		base.MaxDepartmentsPerCollege = *sub.MaxDepartmentsOverride
	}
	if sub.MaxProgramsOverride != nil {
		base.MaxProgramsPerDepartment = *sub.MaxProgramsOverride
	}
	if sub.MaxStudentsOverride != nil {
		base.MaxStudentsPerProgram = *sub.MaxStudentsOverride
	}
	if sub.MaxApplicationsOverride != nil {
		base.MaxApplicationsPerUser = *sub.MaxApplicationsOverride
	}
	if sub.MaxStaffOverride != nil {
		base.MaxStaffUsers = *sub.MaxStaffOverride
	}
	return base
}

// HasOverrides checks if subscription has any overrides.
func HasOverrides(sub *Subscription) bool {
	return sub.MaxCollegesOverride != nil ||
		sub.MaxDepartmentsOverride != nil ||
		sub.MaxProgramsOverride != nil ||
		sub.MaxStudentsOverride != nil ||
		sub.MaxApplicationsOverride != nil ||
		sub.MaxStaffOverride != nil
}

// IsExpired checks if the subscription has expired.
func IsExpired(expiresAt *time.Time) bool {
	if expiresAt == nil {
		return false
	}
	return time.Now().After(*expiresAt)
}

// IsValidTier checks if the tier is valid.
func IsValidTier(tier string) bool {
	switch tier {
	case TierFree, TierBasic, TierPremium:
		return true
	default:
		return false
	}
}

// CanCreate checks if creation is allowed based on current count and limit.
func CanCreate(currentCount, limit int) bool {
	return currentCount < limit
}

// Remaining returns remaining quota.
func Remaining(currentCount, limit int) int {
	r := limit - currentCount
	if r < 0 {
		return 0
	}
	return r
}
