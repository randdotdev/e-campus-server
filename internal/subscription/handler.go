package subscription

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/randdotdev/e-campus-server/internal/middleware"
	"github.com/randdotdev/e-campus-server/internal/authz"
	"github.com/randdotdev/e-campus-server/internal/response"
	"go.uber.org/zap"
)

type Handler struct {
	service *Service
	log     *zap.Logger
}

func NewHandler(service *Service, log *zap.Logger) *Handler {
	return &Handler{
		service: service,
		log:     log,
	}
}

// University admin handlers (read-only)

func (h *Handler) GetMyLimits(c *gin.Context) {
	if !authz.Check(c, authz.ResourceSubscription, authz.ActionGet) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	limits, err := h.service.GetLimits(c.Request.Context())
	if err != nil && !errors.Is(err, ErrSubscriptionExpired) {
		h.log.Error("get limits failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, limits)
}

func (h *Handler) GetMySubscription(c *gin.Context) {
	if !authz.Check(c, authz.ResourceSubscription, authz.ActionGet) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	sub, err := h.service.GetSubscription(c.Request.Context())
	if err != nil {
		if errors.Is(err, ErrSubscriptionNotFound) {
			response.NotFound(c, "subscription not found")
			return
		}
		h.log.Error("get subscription failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	limits, _ := h.service.GetLimits(c.Request.Context())
	response.OK(c, ToSubscriptionResponse(sub, limits))
}

// Platform admin handlers

func (h *Handler) GetSubscription(c *gin.Context) {
	if !authz.Check(c, authz.ResourceSubscription, authz.ActionGet) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	sub, err := h.service.GetSubscription(c.Request.Context())
	if err != nil {
		if errors.Is(err, ErrSubscriptionNotFound) {
			response.NotFound(c, "subscription not found")
			return
		}
		h.log.Error("get subscription failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	limits, _ := h.service.GetLimits(c.Request.Context())
	response.OK(c, ToSubscriptionResponse(sub, limits))
}

func (h *Handler) GetLimits(c *gin.Context) {
	if !authz.Check(c, authz.ResourceSubscription, authz.ActionGet) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	limits, err := h.service.GetLimits(c.Request.Context())
	if err != nil && !errors.Is(err, ErrSubscriptionExpired) {
		h.log.Error("get limits failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, limits)
}

func (h *Handler) GetAllTierLimits(c *gin.Context) {
	if !authz.Check(c, authz.ResourceSubscription, authz.ActionGet) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	tiers, err := h.service.GetAllTierLimits(c.Request.Context())
	if err != nil {
		h.log.Error("get tier limits failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, ToTierLimitsResponses(tiers))
}

func (h *Handler) UpdateTierLimits(c *gin.Context) {
	if !authz.Check(c, authz.ResourceSubscription, authz.ActionUpdate) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	tier := c.Param("tier")
	if !IsValidTier(tier) {
		response.BadRequest(c, "invalid tier")
		return
	}

	var req UpdateTierLimitsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	tl := &TierLimits{
		Tier:                     tier,
		MaxColleges:              req.MaxColleges,
		MaxDepartmentsPerCollege: req.MaxDepartmentsPerCollege,
		MaxProgramsPerDepartment: req.MaxProgramsPerDepartment,
		MaxStudentsPerProgram:    req.MaxStudentsPerProgram,
		MaxApplicationsPerUser:   req.MaxApplicationsPerUser,
		MaxStaffUsers:            req.MaxStaffUsers,
	}

	if err := h.service.UpdateTierLimits(c.Request.Context(), tl); err != nil {
		h.log.Error("update tier limits failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, ToTierLimitsResponse(tl))
}

func (h *Handler) UpdateTier(c *gin.Context) {
	if !authz.Check(c, authz.ResourceSubscription, authz.ActionUpdate) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	var req UpdateTierRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	userID := middleware.GetUserID(c)

	sub, err := h.service.UpdateTier(c.Request.Context(), req.Tier, req.Reason, userID)
	if err != nil {
		if errors.Is(err, ErrInvalidTier) {
			response.BadRequest(c, "invalid tier")
			return
		}
		h.log.Error("update tier failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	limits, _ := h.service.GetLimits(c.Request.Context())
	response.OK(c, ToSubscriptionResponse(sub, limits))
}

func (h *Handler) SetOverrides(c *gin.Context) {
	if !authz.Check(c, authz.ResourceSubscription, authz.ActionUpdate) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	var req SetOverridesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	userID := middleware.GetUserID(c)

	sub, err := h.service.SetOverrides(c.Request.Context(), req.ToOverrides(), req.Reason, userID)
	if err != nil {
		h.log.Error("set overrides failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	limits, _ := h.service.GetLimits(c.Request.Context())
	response.OK(c, ToSubscriptionResponse(sub, limits))
}

func (h *Handler) ClearOverrides(c *gin.Context) {
	if !authz.Check(c, authz.ResourceSubscription, authz.ActionUpdate) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	userID := middleware.GetUserID(c)

	sub, err := h.service.ClearOverrides(c.Request.Context(), "cleared by admin", userID)
	if err != nil {
		h.log.Error("clear overrides failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	limits, _ := h.service.GetLimits(c.Request.Context())
	response.OK(c, ToSubscriptionResponse(sub, limits))
}

func (h *Handler) GetHistory(c *gin.Context) {
	if !authz.Check(c, authz.ResourceSubscription, authz.ActionGet) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	history, err := h.service.GetHistory(c.Request.Context(), limit)
	if err != nil {
		h.log.Error("get history failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, ToHistoriesResponse(history))
}
