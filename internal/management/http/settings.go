package http

import (
	"errors"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/randdotdev/e-campus-server/internal/authz"
	"github.com/randdotdev/e-campus-server/internal/management"
	"github.com/randdotdev/e-campus-server/internal/shared/middleware"
	"github.com/randdotdev/e-campus-server/internal/shared/response"
)

// ── Settings DTOs ─────────────────────────────────────────────────────────────

// UpdateInstitutionRequest binds an institution-section patch.
type UpdateInstitutionRequest struct {
	Name          map[string]string `json:"name" binding:"required"`
	Type          string            `json:"type" binding:"required,oneof=public private"`
	Country       string            `json:"country" binding:"required"`
	Region        string            `json:"region"`
	Accreditation string            `json:"accreditation"`
	Founded       int               `json:"founded"`
	About         map[string]string `json:"about"`
	Address       string            `json:"address"`
	Phone         string            `json:"phone"`
	Email         string            `json:"email"`
	Website       string            `json:"website"`
	LogoURL       string            `json:"logo_url"`
}

// UpdateDegreeLabelRequest binds one degree label.
type UpdateDegreeLabelRequest struct {
	EN    string `json:"en" binding:"required"`
	Local string `json:"local"`
}

// UpdateGradingRequest binds a grading-section patch.
type UpdateGradingRequest struct {
	Display string         `json:"display" binding:"required,oneof=numeric letter both"`
	Scale   map[string]int `json:"scale" binding:"required"`
}

// UpdateFeaturesRequest binds a feature-flags patch.
type UpdateFeaturesRequest struct {
	CreditsTracking       bool `json:"credits_tracking"`
	AllowRetake           bool `json:"allow_retake"`
	AllowPretake          bool `json:"allow_pretake"`
	FullYearRepeat        bool `json:"full_year_repeat"`
	GradeVisibility       bool `json:"grade_visibility"`
	ShowUniversityMembers bool `json:"show_university_members"`
	ShowCourseMembers     bool `json:"show_course_members"`
}

// UpdateAcademicConfigRequest binds an academic-section patch.
type UpdateAcademicConfigRequest struct {
	SemestersPerYear  int    `json:"semesters_per_year" binding:"required,min=1,max=3"`
	MaxFailureRepeats int    `json:"max_failure_repeats" binding:"min=0"`
	DefaultLanguage   string `json:"default_language" binding:"required,oneof=en ku ar"`
}

// UpdateSettingsRequest binds a partial settings update; absent sections
// stay unchanged.
type UpdateSettingsRequest struct {
	Institution  *UpdateInstitutionRequest           `json:"institution"`
	DegreeLabels map[string]UpdateDegreeLabelRequest `json:"degree_labels"`
	Grading      *UpdateGradingRequest               `json:"grading"`
	Features     *UpdateFeaturesRequest              `json:"features"`
	Academic     *UpdateAcademicConfigRequest        `json:"academic"`
}

// SettingsResponse is the settings document's JSON shape.
type SettingsResponse struct {
	Institution  management.Institution            `json:"institution"`
	DegreeLabels map[string]management.DegreeLabel `json:"degree_labels"`
	Grading      management.GradingConfig          `json:"grading"`
	Features     management.Features               `json:"features"`
	Academic     management.AcademicConfig         `json:"academic"`
}

// InstitutionPublicResponse is the public about-page shape (single
// language).
type InstitutionPublicResponse struct {
	Name          string `json:"name"`
	Type          string `json:"type"`
	Country       string `json:"country"`
	Region        string `json:"region,omitempty"`
	Accreditation string `json:"accreditation,omitempty"`
	Founded       int    `json:"founded,omitempty"`
	About         string `json:"about,omitempty"`
	Address       string `json:"address,omitempty"`
	Phone         string `json:"phone,omitempty"`
	Email         string `json:"email,omitempty"`
	Website       string `json:"website,omitempty"`
	LogoURL       string `json:"logo_url,omitempty"`
}

// FeaturesResponse is the feature flags' JSON shape.
type FeaturesResponse = management.Features

func toSettingsResponse(s *management.UniversitySettings) SettingsResponse {
	return SettingsResponse{
		Institution:  s.Institution,
		DegreeLabels: s.DegreeLabels,
		Grading:      s.Grading,
		Features:     s.Features,
		Academic:     s.Academic,
	}
}

func toInstitutionPublicResponse(i management.Institution, lang string) InstitutionPublicResponse {
	return InstitutionPublicResponse{
		Name:          i.GetName(lang),
		Type:          i.Type,
		Country:       i.Country,
		Region:        i.Region,
		Accreditation: i.Accreditation,
		Founded:       i.Founded,
		About:         i.GetAbout(lang),
		Address:       i.Address,
		Phone:         i.Phone,
		Email:         i.Email,
		Website:       i.Website,
		LogoURL:       i.LogoURL,
	}
}

func toSettingsUpdates(req UpdateSettingsRequest) management.SettingsUpdates {
	var updates management.SettingsUpdates

	if req.Institution != nil {
		updates.Institution = &management.Institution{
			Name:          req.Institution.Name,
			Type:          req.Institution.Type,
			Country:       req.Institution.Country,
			Region:        req.Institution.Region,
			Accreditation: req.Institution.Accreditation,
			Founded:       req.Institution.Founded,
			About:         req.Institution.About,
			Address:       req.Institution.Address,
			Phone:         req.Institution.Phone,
			Email:         req.Institution.Email,
			Website:       req.Institution.Website,
			LogoURL:       req.Institution.LogoURL,
		}
	}
	if req.DegreeLabels != nil {
		updates.DegreeLabels = make(map[string]management.DegreeLabel, len(req.DegreeLabels))
		for k, v := range req.DegreeLabels {
			updates.DegreeLabels[k] = management.DegreeLabel(v)
		}
	}
	if req.Grading != nil {
		updates.Grading = &management.GradingConfig{
			Display: req.Grading.Display,
			Scale:   req.Grading.Scale,
		}
	}
	if req.Features != nil {
		updates.Features = &management.Features{
			CreditsTracking:       req.Features.CreditsTracking,
			AllowRetake:           req.Features.AllowRetake,
			AllowPretake:          req.Features.AllowPretake,
			FullYearRepeat:        req.Features.FullYearRepeat,
			GradeVisibility:       req.Features.GradeVisibility,
			ShowUniversityMembers: req.Features.ShowUniversityMembers,
			ShowCourseMembers:     req.Features.ShowCourseMembers,
		}
	}
	if req.Academic != nil {
		updates.Academic = &management.AcademicConfig{
			SemestersPerYear:  req.Academic.SemestersPerYear,
			MaxFailureRepeats: req.Academic.MaxFailureRepeats,
			DefaultLanguage:   req.Academic.DefaultLanguage,
		}
	}
	return updates
}

// canUpdateSettings requires super_admin at university scope or above.
func canUpdateSettings(c *gin.Context) bool {
	role := middleware.GetUserRole(c)
	if role == nil {
		return false
	}
	return role.Level == string(authz.LevelSuperAdmin) &&
		(role.ScopeType == string(authz.ScopeUniversity) || role.ScopeType == string(authz.ScopePlatform))
}

// ── Handlers ──────────────────────────────────────────────────────────────────

// GetSettings handles GET /settings.
func (h *Handler) GetSettings(c *gin.Context) {
	s, err := h.settings.Get(c.Request.Context())
	if errors.Is(err, management.ErrSettingsNotFound) {
		response.NotFound(c, "settings not found")
		return
	}
	if err != nil {
		h.log.Error("get settings failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	response.OK(c, toSettingsResponse(s))
}

// UpdateSettings handles PUT /settings.
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
	result, err := h.settings.UpdatePartial(c.Request.Context(), toSettingsUpdates(req), userID)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, toSettingsResponse(result))
}

// GetInstitution handles GET /settings/institution.
func (h *Handler) GetInstitution(c *gin.Context) {
	s, err := h.settings.Get(c.Request.Context())
	if err != nil {
		h.log.Error("get institution failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	response.OK(c, s.Institution)
}

// UpdateInstitution handles PUT /settings/institution.
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
	updates := management.SettingsUpdates{
		Institution: &management.Institution{
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

	result, err := h.settings.UpdatePartial(c.Request.Context(), updates, userID)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, result.Institution)
}

// GetFeatures handles GET /settings/features.
func (h *Handler) GetFeatures(c *gin.Context) {
	features, err := h.settings.GetFeatures(c.Request.Context())
	if err != nil {
		h.log.Error("get features failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	response.OK(c, features)
}

// UpdateFeatures handles PUT /settings/features.
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
	updates := management.SettingsUpdates{
		Features: &management.Features{
			CreditsTracking:       req.CreditsTracking,
			AllowRetake:           req.AllowRetake,
			AllowPretake:          req.AllowPretake,
			FullYearRepeat:        req.FullYearRepeat,
			GradeVisibility:       req.GradeVisibility,
			ShowUniversityMembers: req.ShowUniversityMembers,
			ShowCourseMembers:     req.ShowCourseMembers,
		},
	}

	result, err := h.settings.UpdatePartial(c.Request.Context(), updates, userID)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, result.Features)
}

// GetPublicAbout handles GET /public/about.
func (h *Handler) GetPublicAbout(c *gin.Context) {
	lang := c.DefaultQuery("lang", "en")
	s, err := h.settings.Get(c.Request.Context())
	if err != nil {
		h.log.Error("get public about failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	response.OK(c, toInstitutionPublicResponse(s.Institution, lang))
}
