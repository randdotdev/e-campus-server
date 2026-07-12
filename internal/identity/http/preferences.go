// Package http is the HTTP transport for the identity domain.
package http

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/randdotdev/e-campus-server/internal/identity"
	"github.com/randdotdev/e-campus-server/internal/shared/middleware"
	"github.com/randdotdev/e-campus-server/internal/shared/response"
)

// PreferencesUpdateRequest is a partial preferences edit; absent fields are
// left unchanged. The oneof lists mirror identity's Valid* predicates.
type PreferencesUpdateRequest struct {
	Language           *identity.Language `json:"language" binding:"omitempty,oneof=en ku"`
	Timezone           *string            `json:"timezone"`
	Theme              *identity.Theme    `json:"theme" binding:"omitempty,oneof=light dark system"`
	EmailNotifications *bool              `json:"email_notifications"`
	PushNotifications  *bool              `json:"push_notifications"`
}

func (r PreferencesUpdateRequest) toUpdates() identity.PreferencesUpdates {
	return identity.PreferencesUpdates{
		Language:           r.Language,
		Timezone:           r.Timezone,
		Theme:              r.Theme,
		EmailNotifications: r.EmailNotifications,
		PushNotifications:  r.PushNotifications,
	}
}

// PreferencesResponse is the wire shape of a user's preferences.
type PreferencesResponse struct {
	Language           identity.Language `json:"language"`
	Timezone           string            `json:"timezone"`
	Theme              identity.Theme    `json:"theme"`
	EmailNotifications bool              `json:"email_notifications"`
	PushNotifications  bool              `json:"push_notifications"`
	UpdatedAt          time.Time         `json:"updated_at"`
}

func toPreferencesResponse(p *identity.UserPreferences) PreferencesResponse {
	return PreferencesResponse{
		Language:           p.Language,
		Timezone:           p.Timezone,
		Theme:              p.Theme,
		EmailNotifications: p.EmailNotifications,
		PushNotifications:  p.PushNotifications,
		UpdatedAt:          p.UpdatedAt,
	}
}

// GetMine returns the caller's preferences.
func (h *Handler) GetMine(c *gin.Context) {
	prefs, err := h.prefs.Get(c.Request.Context(), middleware.GetUserID(c))
	if err != nil {
		h.log.Error("get preferences failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	response.OK(c, toPreferencesResponse(prefs))
}

// UpdateMine applies a partial edit to the caller's preferences.
func (h *Handler) UpdateMine(c *gin.Context) {
	var req PreferencesUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	prefs, err := h.prefs.Update(c.Request.Context(), middleware.GetUserID(c), req.toUpdates())
	if errors.Is(err, identity.ErrInvalidLanguage) {
		response.BadRequest(c, "invalid language")
		return
	}
	if errors.Is(err, identity.ErrInvalidTheme) {
		response.BadRequest(c, "invalid theme")
		return
	}
	if err != nil {
		h.log.Error("update preferences failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	response.OK(c, toPreferencesResponse(prefs))
}
