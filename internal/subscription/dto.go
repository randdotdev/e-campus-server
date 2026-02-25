package subscription

import (
	"time"

	"github.com/google/uuid"
)

// Request DTOs

type UpdateTierRequest struct {
	Tier   string `json:"tier" binding:"required,oneof=free basic premium"`
	Reason string `json:"reason" binding:"required,max=255"`
}

type SetOverridesRequest struct {
	MaxColleges     *int   `json:"max_colleges" binding:"omitempty,min=1"`
	MaxDepartments  *int   `json:"max_departments" binding:"omitempty,min=1"`
	MaxPrograms     *int   `json:"max_programs" binding:"omitempty,min=1"`
	MaxStudents     *int   `json:"max_students" binding:"omitempty,min=1"`
	MaxApplications *int   `json:"max_applications" binding:"omitempty,min=1"`
	MaxStaff        *int   `json:"max_staff" binding:"omitempty,min=1"`
	MaxStorage      *int64 `json:"max_storage" binding:"omitempty,min=1"`
	MaxFileSize     *int64 `json:"max_file_size" binding:"omitempty,min=1"`
	Reason          string `json:"reason" binding:"required,max=255"`
}

func (r SetOverridesRequest) ToOverrides() Overrides {
	return Overrides{
		MaxColleges:     r.MaxColleges,
		MaxDepartments:  r.MaxDepartments,
		MaxPrograms:     r.MaxPrograms,
		MaxStudents:     r.MaxStudents,
		MaxApplications: r.MaxApplications,
		MaxStaff:        r.MaxStaff,
		MaxStorage:      r.MaxStorage,
		MaxFileSize:     r.MaxFileSize,
	}
}

type UpdateTierLimitsRequest struct {
	MaxColleges              int   `json:"max_colleges" binding:"required,min=1"`
	MaxDepartmentsPerCollege int   `json:"max_departments_per_college" binding:"required,min=1"`
	MaxProgramsPerDepartment int   `json:"max_programs_per_department" binding:"required,min=1"`
	MaxStudentsPerProgram    int   `json:"max_students_per_program" binding:"required,min=1"`
	MaxApplicationsPerUser   int   `json:"max_applications_per_user" binding:"required,min=1"`
	MaxStaffUsers            int   `json:"max_staff_users" binding:"required,min=1"`
	MaxStorageBytes          int64 `json:"max_storage_bytes" binding:"required,min=1"`
	MaxFileSizeBytes         int64 `json:"max_file_size_bytes" binding:"required,min=1"`
}

// Response DTOs

type SubscriptionResponse struct {
	ID        uuid.UUID          `json:"id"`
	Tier      string             `json:"tier"`
	Limits    Limits             `json:"limits"`
	Overrides *OverridesResponse `json:"overrides,omitempty"`
	ExpiresAt *time.Time         `json:"expires_at,omitempty"`
	UpdatedBy *uuid.UUID         `json:"updated_by,omitempty"`
	UpdatedAt time.Time          `json:"updated_at"`
}

type OverridesResponse struct {
	MaxColleges     *int   `json:"max_colleges,omitempty"`
	MaxDepartments  *int   `json:"max_departments,omitempty"`
	MaxPrograms     *int   `json:"max_programs,omitempty"`
	MaxStudents     *int   `json:"max_students,omitempty"`
	MaxApplications *int   `json:"max_applications,omitempty"`
	MaxStaff        *int   `json:"max_staff,omitempty"`
	MaxStorage      *int64 `json:"max_storage,omitempty"`
	MaxFileSize     *int64 `json:"max_file_size,omitempty"`
}

type TierLimitsResponse struct {
	Tier                     string    `json:"tier"`
	MaxColleges              int       `json:"max_colleges"`
	MaxDepartmentsPerCollege int       `json:"max_departments_per_college"`
	MaxProgramsPerDepartment int       `json:"max_programs_per_department"`
	MaxStudentsPerProgram    int       `json:"max_students_per_program"`
	MaxApplicationsPerUser   int       `json:"max_applications_per_user"`
	MaxStaffUsers            int       `json:"max_staff_users"`
	MaxStorageBytes          int64     `json:"max_storage_bytes"`
	MaxFileSizeBytes         int64     `json:"max_file_size_bytes"`
	UpdatedAt                time.Time `json:"updated_at"`
}

type HistoryResponse struct {
	ID           uuid.UUID          `json:"id"`
	Tier         string             `json:"tier"`
	Overrides    *OverridesResponse `json:"overrides,omitempty"`
	ExpiresAt    *time.Time         `json:"expires_at,omitempty"`
	ChangedBy    *uuid.UUID         `json:"changed_by,omitempty"`
	ChangedAt    time.Time          `json:"changed_at"`
	ChangeReason *string            `json:"change_reason,omitempty"`
}

// Mapper functions

func ToSubscriptionResponse(sub *Subscription, limits Limits) SubscriptionResponse {
	resp := SubscriptionResponse{
		ID:        sub.ID,
		Tier:      sub.Tier,
		Limits:    limits,
		ExpiresAt: sub.ExpiresAt,
		UpdatedBy: sub.UpdatedBy,
		UpdatedAt: sub.UpdatedAt,
	}

	if HasOverrides(sub) {
		resp.Overrides = &OverridesResponse{
			MaxColleges:     sub.MaxCollegesOverride,
			MaxDepartments:  sub.MaxDepartmentsOverride,
			MaxPrograms:     sub.MaxProgramsOverride,
			MaxStudents:     sub.MaxStudentsOverride,
			MaxApplications: sub.MaxApplicationsOverride,
			MaxStaff:        sub.MaxStaffOverride,
			MaxStorage:      sub.MaxStorageOverride,
			MaxFileSize:     sub.MaxFileSizeOverride,
		}
	}

	return resp
}

func ToTierLimitsResponse(tl *TierLimits) TierLimitsResponse {
	return TierLimitsResponse{
		Tier:                     tl.Tier,
		MaxColleges:              tl.MaxColleges,
		MaxDepartmentsPerCollege: tl.MaxDepartmentsPerCollege,
		MaxProgramsPerDepartment: tl.MaxProgramsPerDepartment,
		MaxStudentsPerProgram:    tl.MaxStudentsPerProgram,
		MaxApplicationsPerUser:   tl.MaxApplicationsPerUser,
		MaxStaffUsers:            tl.MaxStaffUsers,
		MaxStorageBytes:          tl.MaxStorageBytes,
		MaxFileSizeBytes:         tl.MaxFileSizeBytes,
		UpdatedAt:                tl.UpdatedAt,
	}
}

func ToTierLimitsResponses(tiers []TierLimits) []TierLimitsResponse {
	result := make([]TierLimitsResponse, len(tiers))
	for i := range tiers {
		result[i] = ToTierLimitsResponse(&tiers[i])
	}
	return result
}

func ToHistoryResponse(h *History) HistoryResponse {
	resp := HistoryResponse{
		ID:           h.ID,
		Tier:         h.Tier,
		ExpiresAt:    h.ExpiresAt,
		ChangedBy:    h.ChangedBy,
		ChangedAt:    h.ChangedAt,
		ChangeReason: h.ChangeReason,
	}

	if h.MaxCollegesOverride != nil || h.MaxDepartmentsOverride != nil ||
		h.MaxProgramsOverride != nil || h.MaxStudentsOverride != nil ||
		h.MaxApplicationsOverride != nil || h.MaxStaffOverride != nil ||
		h.MaxStorageOverride != nil || h.MaxFileSizeOverride != nil {
		resp.Overrides = &OverridesResponse{
			MaxColleges:     h.MaxCollegesOverride,
			MaxDepartments:  h.MaxDepartmentsOverride,
			MaxPrograms:     h.MaxProgramsOverride,
			MaxStudents:     h.MaxStudentsOverride,
			MaxApplications: h.MaxApplicationsOverride,
			MaxStaff:        h.MaxStaffOverride,
			MaxStorage:      h.MaxStorageOverride,
			MaxFileSize:     h.MaxFileSizeOverride,
		}
	}

	return resp
}

func ToHistoriesResponse(histories []History) []HistoryResponse {
	result := make([]HistoryResponse, len(histories))
	for i := range histories {
		result[i] = ToHistoryResponse(&histories[i])
	}
	return result
}
