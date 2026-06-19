package settings

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/randdotdev/e-campus-server/internal/authz"
	"github.com/randdotdev/e-campus-server/internal/middleware"
	"github.com/randdotdev/e-campus-server/internal/response"
	"go.uber.org/zap"
)

type Handler struct {
	service *Service
	log     *zap.Logger
}

func NewHandler(service *Service, log *zap.Logger) *Handler {
	return &Handler{service: service, log: log}
}

// canUpdateSettings requires super_admin level at university scope or above.
func canUpdateSettings(c *gin.Context) bool {
	role := middleware.GetUserRole(c)
	if role == nil {
		return false
	}
	isSuperAdmin := role.Level == authz.SuperAdmin
	hasSufficientScope := role.ScopeType == authz.ScopeUniversity || role.ScopeType == authz.ScopePlatform
	return isSuperAdmin && hasSufficientScope
}

func (h *Handler) GetSettings(c *gin.Context) {
	settings, err := h.service.Get(c.Request.Context())
	if errors.Is(err, ErrSettingsNotFound) {
		response.NotFound(c, "settings not found")
		return
	}
	if err != nil {
		h.log.Error("get settings failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, ToSettingsResponse(settings))
}

func (h *Handler) UpdateSettings(c *gin.Context) {
	if !canUpdateSettings(c) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	var req UpdateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	userID := middleware.GetUserID(c)
	updates := ToSettingsUpdates(req)

	result, err := h.service.UpdatePartial(c.Request.Context(), updates, userID)
	if errors.Is(err, ErrMissingInstitutionName) {
		response.BadRequest(c, "institution name is required")
		return
	}
	if errors.Is(err, ErrInvalidGradingDisplay) {
		response.BadRequest(c, "invalid grading display mode")
		return
	}
	if errors.Is(err, ErrInvalidSemestersPerYear) {
		response.BadRequest(c, "semesters per year must be 1, 2, or 3")
		return
	}
	if err != nil {
		h.log.Error("update settings failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, ToSettingsResponse(result))
}

func (h *Handler) GetInstitution(c *gin.Context) {
	settings, err := h.service.Get(c.Request.Context())
	if err != nil {
		h.log.Error("get institution failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, settings.Institution)
}

func (h *Handler) UpdateInstitution(c *gin.Context) {
	if !canUpdateSettings(c) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	var req UpdateInstitutionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	userID := middleware.GetUserID(c)
	updates := SettingsUpdates{
		Institution: &Institution{
			Name:          req.Name,
			Type:          req.Type,
			Country:       req.Country,
			Region:        req.Region,
			Accreditation: req.Accreditation,
			Founded:       req.Founded,
			About:         req.About,
			Address:       req.Address,
			Phone:         req.Phone,
			Email:         req.Email,
			Website:       req.Website,
			LogoURL:       req.LogoURL,
		},
	}

	result, err := h.service.UpdatePartial(c.Request.Context(), updates, userID)
	if err != nil {
		h.log.Error("update institution failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, result.Institution)
}

func (h *Handler) GetFeatures(c *gin.Context) {
	features, err := h.service.GetFeatures(c.Request.Context())
	if err != nil {
		h.log.Error("get features failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, ToFeaturesResponse(features))
}

func (h *Handler) UpdateFeatures(c *gin.Context) {
	if !canUpdateSettings(c) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	var req UpdateFeaturesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	userID := middleware.GetUserID(c)
	updates := SettingsUpdates{
		Features: &Features{
			CreditsTracking:       req.CreditsTracking,
			AllowRetake:           req.AllowRetake,
			AllowPretake:          req.AllowPretake,
			FullYearRepeat:        req.FullYearRepeat,
			GradeVisibility:       req.GradeVisibility,
			ShowUniversityMembers: req.ShowUniversityMembers,
			ShowCourseMembers:     req.ShowCourseMembers,
		},
	}

	result, err := h.service.UpdatePartial(c.Request.Context(), updates, userID)
	if err != nil {
		h.log.Error("update features failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, ToFeaturesResponse(result.Features))
}

func (h *Handler) GetPublicAbout(c *gin.Context) {
	lang := c.DefaultQuery("lang", "en")

	settings, err := h.service.Get(c.Request.Context())
	if err != nil {
		h.log.Error("get public about failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, ToInstitutionPublicResponse(settings.Institution, lang))
}
