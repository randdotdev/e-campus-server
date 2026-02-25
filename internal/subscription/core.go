package subscription

import "time"

type Limits struct {
	MaxColleges              int
	MaxDepartmentsPerCollege int
	MaxProgramsPerDepartment int
	MaxStudentsPerProgram    int
	MaxApplicationsPerUser   int
	MaxStaffUsers            int
	MaxStorageBytes          int64
	MaxFileSizeBytes         int64
}

func ToLimits(tl *TierLimits) Limits {
	return Limits{
		MaxColleges:              tl.MaxColleges,
		MaxDepartmentsPerCollege: tl.MaxDepartmentsPerCollege,
		MaxProgramsPerDepartment: tl.MaxProgramsPerDepartment,
		MaxStudentsPerProgram:    tl.MaxStudentsPerProgram,
		MaxApplicationsPerUser:   tl.MaxApplicationsPerUser,
		MaxStaffUsers:            tl.MaxStaffUsers,
		MaxStorageBytes:          tl.MaxStorageBytes,
		MaxFileSizeBytes:         tl.MaxFileSizeBytes,
	}
}

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
	if sub.MaxStorageOverride != nil {
		base.MaxStorageBytes = *sub.MaxStorageOverride
	}
	if sub.MaxFileSizeOverride != nil {
		base.MaxFileSizeBytes = *sub.MaxFileSizeOverride
	}
	return base
}

func HasOverrides(sub *Subscription) bool {
	return sub.MaxCollegesOverride != nil ||
		sub.MaxDepartmentsOverride != nil ||
		sub.MaxProgramsOverride != nil ||
		sub.MaxStudentsOverride != nil ||
		sub.MaxApplicationsOverride != nil ||
		sub.MaxStaffOverride != nil ||
		sub.MaxStorageOverride != nil ||
		sub.MaxFileSizeOverride != nil
}

func IsExpired(expiresAt *time.Time) bool {
	if expiresAt == nil {
		return false
	}
	return time.Now().After(*expiresAt)
}

func IsValidTier(tier string) bool {
	switch tier {
	case TierFree, TierBasic, TierPremium:
		return true
	default:
		return false
	}
}

func CanCreate(currentCount, limit int) bool {
	return currentCount < limit
}

func Remaining(currentCount, limit int) int {
	r := limit - currentCount
	if r < 0 {
		return 0
	}
	return r
}

type Overrides struct {
	MaxColleges     *int
	MaxDepartments  *int
	MaxPrograms     *int
	MaxStudents     *int
	MaxApplications *int
	MaxStaff        *int
	MaxStorage      *int64
	MaxFileSize     *int64
}

func SetOverridesOnSubscription(sub *Subscription, overrides Overrides) *Subscription {
	result := *sub
	if overrides.MaxColleges != nil {
		result.MaxCollegesOverride = overrides.MaxColleges
	}
	if overrides.MaxDepartments != nil {
		result.MaxDepartmentsOverride = overrides.MaxDepartments
	}
	if overrides.MaxPrograms != nil {
		result.MaxProgramsOverride = overrides.MaxPrograms
	}
	if overrides.MaxStudents != nil {
		result.MaxStudentsOverride = overrides.MaxStudents
	}
	if overrides.MaxApplications != nil {
		result.MaxApplicationsOverride = overrides.MaxApplications
	}
	if overrides.MaxStaff != nil {
		result.MaxStaffOverride = overrides.MaxStaff
	}
	if overrides.MaxStorage != nil {
		result.MaxStorageOverride = overrides.MaxStorage
	}
	if overrides.MaxFileSize != nil {
		result.MaxFileSizeOverride = overrides.MaxFileSize
	}
	return &result
}

func ClearOverridesOnSubscription(sub *Subscription) *Subscription {
	result := *sub
	result.MaxCollegesOverride = nil
	result.MaxDepartmentsOverride = nil
	result.MaxProgramsOverride = nil
	result.MaxStudentsOverride = nil
	result.MaxApplicationsOverride = nil
	result.MaxStaffOverride = nil
	result.MaxStorageOverride = nil
	result.MaxFileSizeOverride = nil
	return &result
}

func DefaultHistoryLimit(limit int) int {
	if limit <= 0 {
		return 50
	}
	return limit
}
