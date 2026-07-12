package http

import (
	"errors"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/shared/middleware"
	"github.com/randdotdev/e-campus-server/internal/shared/response"
	"github.com/randdotdev/e-campus-server/internal/subscription"
)

// ── Request DTOs ─────────────────────────────────────────────────────────────

// UpdateTierRequest binds a tier change.
type UpdateTierRequest struct {
	Tier   string `json:"tier" binding:"required,oneof=free basic premium"`
	Reason string `json:"reason" binding:"required,max=255"`
}

// SetOverridesRequest binds a partial set of per-institution limit overrides.
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

func (r SetOverridesRequest) toOverrides() subscription.Overrides {
	return subscription.Overrides{
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

// UpdateTierLimitsRequest binds a tier's limit table row.
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

// ── Response DTOs ────────────────────────────────────────────────────────────

// SubscriptionResponse is the subscription's JSON shape with its effective
// limits.
type SubscriptionResponse struct {
	ID        uuid.UUID           `json:"id"`
	Tier      subscription.Tier   `json:"tier"`
	Limits    subscription.Limits `json:"limits"`
	Overrides *OverridesResponse  `json:"overrides,omitempty"`
	ExpiresAt *time.Time          `json:"expires_at,omitempty"`
	UpdatedBy *uuid.UUID          `json:"updated_by,omitempty"`
	UpdatedAt time.Time           `json:"updated_at"`
}

// OverridesResponse is the per-institution overrides' JSON shape.
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

// TierLimitsResponse is one tier's limit table row.
type TierLimitsResponse struct {
	Tier                     subscription.Tier `json:"tier"`
	MaxColleges              int               `json:"max_colleges"`
	MaxDepartmentsPerCollege int               `json:"max_departments_per_college"`
	MaxProgramsPerDepartment int               `json:"max_programs_per_department"`
	MaxStudentsPerProgram    int               `json:"max_students_per_program"`
	MaxApplicationsPerUser   int               `json:"max_applications_per_user"`
	MaxStaffUsers            int               `json:"max_staff_users"`
	MaxStorageBytes          int64             `json:"max_storage_bytes"`
	MaxFileSizeBytes         int64             `json:"max_file_size_bytes"`
	UpdatedAt                time.Time         `json:"updated_at"`
}

// HistoryResponse is one subscription change record.
type HistoryResponse struct {
	ID           uuid.UUID          `json:"id"`
	Tier         subscription.Tier  `json:"tier"`
	Overrides    *OverridesResponse `json:"overrides,omitempty"`
	ExpiresAt    *time.Time         `json:"expires_at,omitempty"`
	ChangedBy    *uuid.UUID         `json:"changed_by,omitempty"`
	ChangedAt    time.Time          `json:"changed_at"`
	ChangeReason *string            `json:"change_reason,omitempty"`
}

func toSubscriptionResponse(sub *subscription.Subscription, limits subscription.Limits) SubscriptionResponse {
	resp := SubscriptionResponse{
		ID:        sub.ID,
		Tier:      sub.Tier,
		Limits:    limits,
		ExpiresAt: sub.ExpiresAt,
		UpdatedBy: sub.UpdatedBy,
		UpdatedAt: sub.UpdatedAt,
	}
	if subscription.HasOverrides(sub) {
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

func toTierLimitsResponse(tl *subscription.TierLimits) TierLimitsResponse {
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

func toTierLimitsResponses(tiers []subscription.TierLimits) []TierLimitsResponse {
	result := make([]TierLimitsResponse, len(tiers))
	for i := range tiers {
		result[i] = toTierLimitsResponse(&tiers[i])
	}
	return result
}

func toHistoryResponse(h *subscription.History) HistoryResponse {
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

func toHistoriesResponse(histories []subscription.History) []HistoryResponse {
	result := make([]HistoryResponse, len(histories))
	for i := range histories {
		result[i] = toHistoryResponse(&histories[i])
	}
	return result
}

// ── Handlers ──────────────────────────────────────────────────────────────────

// GetSubscription handles GET /subscription.
func (h *Handler) GetSubscription(c *gin.Context) {
	sub, err := h.svc.GetSubscription(c.Request.Context())
	if err != nil {
		h.respondError(c, err)
		return
	}
	limits, _ := h.svc.GetLimits(c.Request.Context())
	response.OK(c, toSubscriptionResponse(sub, limits))
}

// GetLimits handles GET /subscription/limits. An expired subscription still
// answers with the free-tier limits rather than an error.
func (h *Handler) GetLimits(c *gin.Context) {
	limits, err := h.svc.GetLimits(c.Request.Context())
	if err != nil && !errors.Is(err, subscription.ErrSubscriptionExpired) {
		h.respondError(c, err)
		return
	}
	response.OK(c, limits)
}

// GetAllTierLimits handles GET /platform/tier-limits.
func (h *Handler) GetAllTierLimits(c *gin.Context) {
	tiers, err := h.svc.GetAllTierLimits(c.Request.Context())
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, toTierLimitsResponses(tiers))
}

// UpdateTierLimits handles PUT /platform/tier-limits/:tier.
func (h *Handler) UpdateTierLimits(c *gin.Context) {
	tier := subscription.Tier(c.Param("tier"))
	if !subscription.ValidTier(tier) {
		response.BadRequest(c, "invalid tier")
		return
	}
	var req UpdateTierLimitsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	tl := &subscription.TierLimits{
		Tier:                     tier,
		MaxColleges:              req.MaxColleges,
		MaxDepartmentsPerCollege: req.MaxDepartmentsPerCollege,
		MaxProgramsPerDepartment: req.MaxProgramsPerDepartment,
		MaxStudentsPerProgram:    req.MaxStudentsPerProgram,
		MaxApplicationsPerUser:   req.MaxApplicationsPerUser,
		MaxStaffUsers:            req.MaxStaffUsers,
		MaxStorageBytes:          req.MaxStorageBytes,
		MaxFileSizeBytes:         req.MaxFileSizeBytes,
	}
	if err := h.svc.UpdateTierLimits(c.Request.Context(), tl); err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, toTierLimitsResponse(tl))
}

// UpdateTier handles PUT /platform/subscription/tier.
func (h *Handler) UpdateTier(c *gin.Context) {
	var req UpdateTierRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	sub, err := h.svc.UpdateTier(c.Request.Context(), subscription.Tier(req.Tier), req.Reason, middleware.GetUserID(c))
	if err != nil {
		h.respondError(c, err)
		return
	}
	limits, _ := h.svc.GetLimits(c.Request.Context())
	response.OK(c, toSubscriptionResponse(sub, limits))
}

// SetOverrides handles PUT /platform/subscription/overrides.
func (h *Handler) SetOverrides(c *gin.Context) {
	var req SetOverridesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	sub, err := h.svc.SetOverrides(c.Request.Context(), req.toOverrides(), req.Reason, middleware.GetUserID(c))
	if err != nil {
		h.respondError(c, err)
		return
	}
	limits, _ := h.svc.GetLimits(c.Request.Context())
	response.OK(c, toSubscriptionResponse(sub, limits))
}

// ClearOverrides handles DELETE /platform/subscription/overrides.
func (h *Handler) ClearOverrides(c *gin.Context) {
	sub, err := h.svc.ClearOverrides(c.Request.Context(), "cleared by admin", middleware.GetUserID(c))
	if err != nil {
		h.respondError(c, err)
		return
	}
	limits, _ := h.svc.GetLimits(c.Request.Context())
	response.OK(c, toSubscriptionResponse(sub, limits))
}

// GetHistory handles GET /platform/subscription/history.
func (h *Handler) GetHistory(c *gin.Context) {
	limit := 0
	if l := c.Query("limit"); l != "" {
		parsed, err := strconv.Atoi(l)
		if err != nil || parsed < 1 {
			response.BadRequest(c, "invalid limit")
			return
		}
		limit = parsed
	}
	history, err := h.svc.GetHistory(c.Request.Context(), limit)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, toHistoriesResponse(history))
}
