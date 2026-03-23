package settings

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/ranjdotdev/e-campus-server/internal/middleware"
	"github.com/ranjdotdev/e-campus-server/internal/permission"
	"github.com/ranjdotdev/e-campus-server/internal/response"
	"go.uber.org/zap"
)

type Handler struct {
	svc *Service
	log *zap.Logger
}

func NewHandler(svc *Service, log *zap.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

func (h *Handler) GetSettings(c *gin.Context) {
	settings, err := h.svc.Get(c.Request.Context())
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
	if !permission.CanAdminUniversity(c) {
		response.Forbidden(c, "university admin access required")
		return
	}

	var req UpdateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	userID := middleware.GetUserID(c)
	updates := ToSettingsUpdates(req)

	result, err := h.svc.UpdatePartial(c.Request.Context(), updates, userID)
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
	settings, err := h.svc.Get(c.Request.Context())
	if err != nil {
		h.log.Error("get institution failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, settings.Institution)
}

func (h *Handler) UpdateInstitution(c *gin.Context) {
	if !permission.CanAdminUniversity(c) {
		response.Forbidden(c, "university admin access required")
		return
	}

	var req UpdateInstitutionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	userID := middleware.GetUserID(c)
	updates := SettingsUpdates{
		Institution: &Institution{
			NameEN:  req.NameEN,
			NameKU:  req.NameKU,
			NameAR:  req.NameAR,
			Type:    req.Type,
			Country: req.Country,
			Region:  req.Region,
		},
	}

	result, err := h.svc.UpdatePartial(c.Request.Context(), updates, userID)
	if err != nil {
		h.log.Error("update institution failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, result.Institution)
}

func (h *Handler) GetFeatures(c *gin.Context) {
	features, err := h.svc.GetFeatures(c.Request.Context())
	if err != nil {
		h.log.Error("get features failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, ToFeaturesResponse(features))
}

func (h *Handler) UpdateFeatures(c *gin.Context) {
	if !permission.CanAdminUniversity(c) {
		response.Forbidden(c, "university admin access required")
		return
	}

	var req UpdateFeaturesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	userID := middleware.GetUserID(c)
	updates := SettingsUpdates{
		Features: &Features{
			CreditsTracking: req.CreditsTracking,
			AllowRetake:     req.AllowRetake,
			AllowPretake:    req.AllowPretake,
			FullYearRepeat:  req.FullYearRepeat,
			GradeVisibility: req.GradeVisibility,
		},
	}

	result, err := h.svc.UpdatePartial(c.Request.Context(), updates, userID)
	if err != nil {
		h.log.Error("update features failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, ToFeaturesResponse(result.Features))
}

func (h *Handler) GetMyPreferences(c *gin.Context) {
	userID := middleware.GetUserID(c)

	prefs, err := h.svc.GetPreferences(c.Request.Context(), userID)
	if err != nil {
		h.log.Error("get preferences failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, ToPreferencesResponse(prefs))
}

func (h *Handler) UpdateMyPreferences(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req UpdatePreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	updates := ToPreferencesUpdates(req)
	prefs, err := h.svc.UpdatePreferences(c.Request.Context(), userID, updates)
	if errors.Is(err, ErrInvalidLanguage) {
		response.BadRequest(c, "invalid language")
		return
	}
	if err != nil {
		h.log.Error("update preferences failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, ToPreferencesResponse(prefs))
}
